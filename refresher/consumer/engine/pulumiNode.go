package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/infralight/go-kit/db/mongo"
	"github.com/infralight/go-kit/flywheel/arn"
	"github.com/infralight/go-kit/helpers"
	k8sUtils "github.com/infralight/go-kit/k8s"
	goKit "github.com/infralight/go-kit/pulumi"
	goKitTypes "github.com/infralight/go-kit/types"
	k8sApiUtils "github.com/infralight/k8s-api/pkg/utils"
	"github.com/infralight/pulumi/refresher"
	"github.com/infralight/pulumi/refresher/config"
	"github.com/infralight/pulumi/refresher/utils"
	"github.com/pulumi/pulumi/pkg/v3/engine"
	"github.com/pulumi/pulumi/pkg/v3/resource/deploy"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/rs/zerolog"
	"github.com/thoas/go-funk"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"time"
)

func CreatePulumiNodes(events []engine.Event, accountId, stackId, integrationId, stackName, projectName, organizationName string, logger *zerolog.Logger, config *config.Config) (result []PulumiNode, assetTypes []string, err error) {

	var nodes []PulumiNode
	var k8sNodes []PulumiNode
	var uids []string
	var kinds []string
	awsCommonProviders := make(map[string]int)
	k8sCommonProviders := make(map[string]int)
	ctx := context.Background()

	awsIntegrations, err := utils.ListAwsIntegrations(ctx, config, accountId, logger)
	if err != nil {
		logger.Err(err).Str("accountId", accountId).Str("integrationId", integrationId).Msg("failed to list aws integrations")
		return nil, nil, err
	}
	k8sIntegrations, err := utils.ListK8sIntegrations(ctx, config, accountId, logger)
	if err != nil {
		logger.Err(err).Str("accountId", accountId).Str("integrationId", integrationId).Msg("failed to list k8s integrations")
		return nil, nil, err
	}

	stack, err := utils.GetStack(ctx, config, accountId, stackId, logger)
	if err != nil {
		logger.Err(err).Str("accountId", accountId).Str("integrationId", integrationId).Msg("failed to get stack")
		return nil, nil, err
	}

	for _, event := range events {
		var metadata = getSameMetadata(event)
		var state engine.StepEventStateMetadata

		var node PulumiNode
		node.Metadata.StackId = stackId
		node.Metadata.StackName = stackName
		node.Metadata.ProjectName = projectName
		node.Metadata.OrganizationName = organizationName
		node.Metadata.PulumiType = metadata.Type.String()
		node.StackId = stackId
		node.Iac = "pulumi"
		node.AccountId = accountId
		node.PulumiIntegrationId = integrationId
		node.IsOrchestrator = false
		node.UpdatedAt = time.Now().Unix()

		if strings.HasPrefix(metadata.Type.String(), "aws:") {
			node.Type = "aws"
			terraformType, err := goKit.GetTerraformTypeByPulumi(metadata.Type.String())
			if err != nil {
				logger.Warn().Str("pulumiAssetType", metadata.Type.String()).Msg("missing pulumi to terraform type mapping")
			} else {
				node.ObjectType = terraformType
				if !helpers.StringSliceContains(assetTypes, terraformType) {
					assetTypes = append(assetTypes, terraformType)
				}
			}

			switch metadata.Op {
			case deploy.OpSame:
				state = *metadata.New
				node.Metadata.PulumiState = "managed"
			case deploy.OpDelete:
				state = *metadata.Old
				node.Metadata.PulumiState = "ghost"
			case deploy.OpUpdate:
				state = *metadata.New
				drifts, err := refresher.CalcDrift(metadata)
				if err != nil {
					logger.Warn().Err(err).Msg("failed to calc some of the drifts")
				}
				if drifts == nil {
					node.Metadata.PulumiState = "managed"

				} else {
					node.Metadata.PulumiState = "modified"
					node.Metadata.PulumiDrifts = drifts
				}
			default:
				continue
			}
			if len(state.Outputs) == 0 {
				continue
			}
			node.Attributes = getIacAttributes(state.Outputs)
			if ARN := state.Outputs["arn"].V; ARN != nil {
				node.Arn = goKitTypes.ToString(ARN)
				awsAccount, region, err := getAccountAndRegionFromArn(fmt.Sprintf("%v", ARN))
				if err != nil {
					logger.Err(err).Str("accountId", accountId).Str("pulumiIntegrationId", integrationId).
						Str("projectName", projectName).Str("stackName", stackName).
						Str("OrganizationName", organizationName).Interface("arn", ARN).
						Msg("failed to parse arn")
					continue
				}
				node.Region = region
				node.ProviderAccountId = awsAccount
				node.AssetId = node.Arn
				if awsIntegrationId := getAwsIntegrationId(awsIntegrations, awsAccount); awsIntegrationId != "" {
					node.AwsIntegration = awsIntegrationId
				}
				if count, ok := awsCommonProviders[awsAccount]; ok {
					awsCommonProviders[awsAccount] = count + 1
				} else {
					awsCommonProviders[awsAccount] = 1
				}

			} else {
				logger.Warn().Str("accountId", accountId).Str("pulumiIntegrationId", integrationId).
					Str("projectName", projectName).Str("stackName", stackName).
					Str("OrganizationName", organizationName).Str("type", metadata.Type.String()).
					Msg("no arn for resource")
				continue
			}
			nodes = append(nodes, node)

		} else if strings.HasPrefix(metadata.Type.String(), "kubernetes:") {
			node.Type = "k8s"
			// k8s flow currently supports only managed state
			var uid string
			newState := *metadata.New
			node.Attributes , err = getK8sIacAttributes(newState.Outputs, []string{"status", "__inputs", "__initialApiVersion"})
			if err != nil {
				logger.Err(err).Msg("failed to get iac attributes")
				continue
			}
			if resourceMetadata := newState.Outputs["metadata"].Mappable(); resourceMetadata != nil {
				namespace := funk.Get(resourceMetadata, "namespace")
				name := funk.Get(resourceMetadata, "name")
				interfaceUid := funk.Get(resourceMetadata, "uid")
				uid = goKitTypes.ToString(interfaceUid)

				if name == nil || interfaceUid == nil {
					logger.Warn().Err(errors.New("found resource with empty name/uid"))
					continue
				}
				if namespace != nil {
					node.Location = goKitTypes.ToString(namespace)
				} else {
					node.Location = ""
				}
				node.Name = goKitTypes.ToString(name)
				node.ResourceId = goKitTypes.ToString(interfaceUid)

			} else {
				logger.Warn().Str("accountId", accountId).Str("pulumiIntegrationId", integrationId).
					Str("stackId", stackId).Msg("found k8s resource without metadata")
				continue
			}
			var resourceKind interface{}
			if resourceKind = newState.Outputs["kind"].V; resourceKind == nil {
				logger.Warn().Str("accountId", accountId).Str("pulumiIntegrationId", integrationId).
					Str("stackId", stackId).Msg("found k8s resource without kind")
				continue
			}
			kind := goKitTypes.ToString(resourceKind)
			node.ObjectType = k8sUtils.GetKubernetesResourceType(kind, goKitTypes.ToString(node.Name))
			node.Kind = kind
			if !helpers.StringSliceContains(uids, uid) {
				uids = append(uids, uid)
			}

			if !helpers.StringSliceContains(kinds, kind) {
				kinds = append(kinds, kind)
			}
			assetTypes = append(assetTypes, goKitTypes.ToString(node.ObjectType))
			k8sNodes = append(k8sNodes, node)
		}

	}
	if len(k8sNodes) > 0 {
		k8sNodes, clusterId, err := buildK8sArns(k8sNodes, accountId, uids, kinds, config, logger, ctx)
		if err != nil {
			logger.Err(err).Msg("failed to build k8s arns")
		} else {
			nodes = append(nodes, k8sNodes...)
			k8sCommonProviders[clusterId] = 1

		}
	}

	err = handleAwsCommonProviders(ctx, accountId, stackId, awsCommonProviders, stack, awsIntegrations, config)
	if err != nil {
		logger.Err(err).Msg("failed to update aws common provider")
	}
	err = handleK8sCommonProviders(ctx, accountId, stackId, k8sCommonProviders, stack, k8sIntegrations, config)
	if err != nil {
		logger.Err(err).Msg("failed to update k8s common provider")
	}
	return nodes, assetTypes, nil
}

func getSameMetadata(event engine.Event) engine.StepEventMetadata {
	var metadata engine.StepEventMetadata
	if event.Type == engine.ResourcePreEvent {
		metadata = event.Payload().(engine.ResourcePreEventPayload).Metadata

	} else if event.Type == engine.ResourceOutputsEvent {
		metadata = event.Payload().(engine.ResourceOutputsEventPayload).Metadata
	}
	return metadata
}

func getAccountAndRegionFromArn(assetArn string) (account, region string, err error) {
	parsedArn, err := arn.Parse(assetArn)
	if err != nil {
		return "", "", err
	}
	region = parsedArn.Location
	if region == "" {
		region = "global"
	}
	return parsedArn.AccountID, region, nil
}

func getK8sIacAttributes(outputs resource.PropertyMap, blackList []string) (string, error) {
	// in case we want k8s attributes we use blacklist for redundant attributes
	iacAttributes := make(map[string]interface{})
	for key, val := range outputs {
		stringKey := fmt.Sprintf("%v", key)
		if !helpers.StringSliceContains(blackList, stringKey) {
			iacAttributes[stringKey] = val.Mappable()
		}
	}
	if metadata, err := k8sApiUtils.GetMapFromMap(iacAttributes, "metadata"); err == nil {
		_ = k8sApiUtils.ConvertItemToYaml(metadata, "managedFields")

	} else {
		return "", errors.New("failed to get metadata")
	}

	if data, err := k8sApiUtils.GetMapFromMap(iacAttributes, "data"); err == nil {
		if dataSpec, err := k8sApiUtils.GetMapFromMap(data, "spec"); err == nil {
			if err = k8sApiUtils.ConvertItemToYaml(dataSpec, "template"); err != nil {
				return "", errors.New("failed to find or to yaml template")
			}
		}
	}
	attributesBytes, err := json.Marshal(&iacAttributes)
	if err != nil {
		return "", errors.New("failed to find or to marshal attributes")
	}
	return string(attributesBytes), nil
}

func getIacAttributes(outputs resource.PropertyMap) string {
	iacAttributes := make(map[string]interface{})
	for key, val := range outputs {
		stringKey := fmt.Sprintf("%v", key)
		iacAttributes[stringKey] = val.Mappable()
	}

	attributesBytes, err := json.Marshal(&iacAttributes)
	if err != nil {
		return ""
	}
	return string(attributesBytes)
}

func buildK8sArns(k8sNodes []PulumiNode, accountId string, uids, kinds []string, cfg *config.Config, logger *zerolog.Logger, ctx context.Context) ([]PulumiNode, string, error) {
	var clusterId string
	integrationIds, err := utils.GetK8sIntegrationIds(accountId, uids, kinds, logger)
	if err != nil || len(integrationIds) == 0 {
		logger.Err(err).Msg("failed to get k8s integration")
		clusterId = "K8sCluster"
	}
	if len(integrationIds) > 1 {
		return nil, "", errors.New("found more than one k8s integrations")
	}
	if clusterId != "K8sCluster" {
		clusterId, err = utils.GetClusterId(ctx, cfg, integrationIds[0], accountId, logger)
		if err != nil {
			logger.Err(err).Msg("failed to get cluster id")
			return nil, "", err
		}
	}

	k8sNodes = funk.Map(k8sNodes, func(node PulumiNode) PulumiNode {
			node.Arn = k8sUtils.BuildArn(goKitTypes.ToString(node.Location), clusterId, goKitTypes.ToString(node.Kind), goKitTypes.ToString(node.Name))
			node.AssetId = k8sUtils.BuildArn(goKitTypes.ToString(node.Location), clusterId, goKitTypes.ToString(node.Kind), goKitTypes.ToString(node.Name))
			if len(integrationIds) > 0 {
				node.K8sIntegration = integrationIds[0]
		}
		return node
	}).([]PulumiNode)
	return k8sNodes, clusterId, nil
}

func getAwsIntegrationId(awsIntegrations []mongo.AwsIntegration, providerAccountId string) string {
	if awsIntegrations != nil {
		for _, integration := range awsIntegrations {
			if integration.AccountNumber == providerAccountId {
				return integration.ID
			}
		}
	}
	return ""
}

func getK8sIntegrationId(k8sIntegrations []mongo.K8sIntegration, clusterId string) string {
	if k8sIntegrations != nil {
		for _, integration := range k8sIntegrations {
			if integration.ClusterId == clusterId {
				return integration.ID
			}
		}
	}
	return ""
}

func handleAwsCommonProviders(ctx context.Context, accountId, stackId string, commonProviderMap map[string]int, stack *mongo.GlobalStack, awsIntegrations []mongo.AwsIntegration, config *config.Config) error {
	if len(commonProviderMap) != 0 {
		max := 0
		var mostCommonProvider string
		var err error
		updateDict := make(bson.M)
		for providerId, count := range commonProviderMap {
			if count > max {
				mostCommonProvider = providerId
			}
		}
		integrationId := getAwsIntegrationId(awsIntegrations, mostCommonProvider)

		if awsIntegration, ok := stack.Integrations["aws"]; ok {
			if externalId, ok := awsIntegration["externalId"]; ok {
				if externalId != mostCommonProvider {
					updateDict["integrations.aws.externalId"] = externalId
				}
			}
			if awsIntegrationId, ok := awsIntegration["id"]; ok {
				if awsIntegrationId != integrationId && integrationId != "" {
					updateDict["integrations.aws.id"], err = primitive.ObjectIDFromHex(integrationId)
					if err != nil {
						return err
					}
				}
			}
		} else {
			updateDict["integrations.aws.externalId"] = mostCommonProvider
			if integrationId != "" {
				updateDict["integrations.aws.id"] = integrationId
			}
		}

		if len(updateDict) != 0  {
			updateDict["updatedAt"] = time.Now().Format(time.RFC3339)
			err = utils.UpdateStack(ctx, config, accountId, stackId, updateDict)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func handleK8sCommonProviders(ctx context.Context, accountId, stackId string, commonProviderMap map[string]int, stack *mongo.GlobalStack, k8sIntegrations []mongo.K8sIntegration, config *config.Config) error {
	if len(commonProviderMap) != 0 {
		max := 0
		var mostCommonProvider string
		var err error
		updateDict := make(bson.M)
		for providerId, count := range commonProviderMap {
			if count > max {
				mostCommonProvider = providerId
			}
		}
		integrationId := getK8sIntegrationId(k8sIntegrations, mostCommonProvider)

		if k8sIntegration, ok := stack.Integrations["k8s"]; ok {
			if externalId, ok := k8sIntegration["externalId"]; ok {
				if externalId != mostCommonProvider {
					updateDict["integrations.k8s.externalId"] = externalId
				}
			}
			if k8sIntegrationId, ok := k8sIntegration["id"]; ok {
				if k8sIntegrationId != integrationId && integrationId != "" {
					updateDict["integrations.k8s.id"], err = primitive.ObjectIDFromHex(integrationId)
					if err != nil {
						return err
					}
				}
			}
		} else {
			updateDict["integrations.k8s.externalId"] = mostCommonProvider
			if integrationId != "" {
				updateDict["integrations.k8s.id"] = integrationId
			}
		}

		if len(updateDict) != 0 {
			updateDict["updatedAt"] = time.Now().Format(time.RFC3339)
			err = utils.UpdateStack(ctx, config, accountId, stackId, updateDict)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
