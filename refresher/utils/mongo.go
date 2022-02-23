package utils

import (
	"context"
	"fmt"
	"github.com/infralight/go-kit/db/mongo"
	"github.com/infralight/pulumi/refresher/config"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

func GetClusterId(ctx context.Context, cfg *config.Config, integrationId, accountId string, logger *zerolog.Logger) (string, error) {
	client, err := mongo.NewClient(cfg.MongoURI)
	if err != nil {
		logger.Err(err).Msg("failed to create new mongo client")
		return "", err
	}
	integration, err := client.GetK8sIntegration(ctx, integrationId, accountId)
	if err != nil {
		logger.Err(err).Msg("failed to get matching k8s integration ids")
		return "", err
	}
	return integration.ClusterId, nil

}

func ListAwsIntegrations(ctx context.Context, cfg *config.Config, accountId string, logger *zerolog.Logger) ([]mongo.AwsIntegration, error) {
	client, err := mongo.NewClient(cfg.MongoURI)
	if err != nil {
		logger.Err(err).Msg("failed to create new mongo client")
		return nil, err
	}
	awsIntegrations, err := client.ListAWSIntegrationsByAccountId(ctx, accountId)
	if err != nil {
		logger.Err(err).Msg("failed to list account's aws integrations")
		return nil, err
	}
	return awsIntegrations, nil

}

func ListK8sIntegrations(ctx context.Context, cfg *config.Config, accountId string, logger *zerolog.Logger) ([]mongo.K8sIntegration, error) {
	client, err := mongo.NewClient(cfg.MongoURI)
	if err != nil {
		logger.Err(err).Msg("failed to create new mongo client")
		return nil, err
	}
	k8sIntegrations, err := client.ListK8SIntegrations(ctx, accountId)
	if err != nil {
		logger.Err(err).Msg("failed to list account's aws integrations")
		return nil, err
	}
	return k8sIntegrations, nil

}

func GetStack(ctx context.Context, cfg *config.Config, accountId, stackId string, logger *zerolog.Logger) (*mongo.GlobalStack, error) {
	client, err := mongo.NewClient(cfg.MongoURI)
	if err != nil {
		logger.Err(err).Msg("failed to create new mongo client")
		return nil, err
	}
	stack, err := client.GetStack(ctx, accountId, stackId, nil)
	if err != nil {
		logger.Err(err).Str("stackId", stackId).Str("accountId", accountId).Msg("failed to get stack")
		return nil, err
	}
	return stack, nil

}

func UpdateStack(ctx context.Context, cfg *config.Config, accountId, stackId string, updates bson.M) error {

	if updates == nil {
		return fmt.Errorf("no updates found")
	}
	client, err := mongo.NewClient(cfg.MongoURI)
	if err != nil {
		return err
	}

	updates["updatedAt"] = time.Now().Format(time.RFC3339)
	_, err = client.UpdateStack(ctx, accountId, stackId, nil, bson.M{
		"$set": updates,
	})
	if err != nil {
		return err
	}
	return nil

}
