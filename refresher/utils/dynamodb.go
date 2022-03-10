package utils

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	goKitTypes "github.com/infralight/go-kit/types"
	goKitDynamo "github.com/infralight/go-kit/db/dynamo"
	"github.com/infralight/pulumi/refresher/config"
	"strings"
	"time"
)

func BatchGetItems(
	items *dynamodb.KeysAndAttributes,
	tableName string,
) ([]interface{}, error) {

	cfg, err := config.LoadConfig()

	sess := cfg.LoadAwsSession()
	svc := dynamodb.New(sess)

	res, err := svc.BatchGetItem(&dynamodb.BatchGetItemInput{RequestItems: map[string]*dynamodb.KeysAndAttributes{tableName: items}})
	if err != nil {
		return nil, fmt.Errorf("failed to get batch from dynamoDB: %w", err)
	}
	if len(res.Responses) == 0 {
		return nil, nil
	}
	responseItems := make([]interface{}, 0, len(res.Responses[tableName]))
	for _, response := range res.Responses[tableName] {
		var out interface{}
		err = dynamodbattribute.UnmarshalMap(response, &out)
		if err != nil {
			continue
		}
		responseItems = append(responseItems, out)
	}
	return responseItems, nil
}

func BatchPutItems(
	items []*dynamodb.WriteRequest,
	tableName string,
) error {

	cfg, err := config.LoadConfig()

	sess := cfg.LoadAwsSession()
	svc := dynamodb.New(sess)
	_, err = svc.BatchWriteItem(&dynamodb.BatchWriteItemInput{RequestItems: map[string][]*dynamodb.WriteRequest{tableName: items}})
	if err != nil {
		return fmt.Errorf("failed to write batch to dynamoDB: %w", err)
	}
	return nil
}

func GetAtrsFromDynamo(accountId, stackId, awsIntegrationId, k8sIntegrationId, tableName string, assetTypesWithRegions []string) {
	items := dynamodb.KeysAndAttributes{}
	for _, assetWithRegion := range assetTypesWithRegions {
		if strings.HasPrefix(assetWithRegion, "aws") && len(awsIntegrationId) > 0 {
			atr := fmt.Sprintf("%s-%s", awsIntegrationId, assetWithRegion)
			items.Keys = append(items.Keys, map[string]*dynamodb.AttributeValue{
				"AccountId": {
					S: aws.String(accountId),
				},
				"ATR": {
					S: aws.String(atr),
				},
			})
		} else if strings.HasPrefix(assetWithRegion, "kubernetes") {
			atr := fmt.Sprintf("%s-%s", k8sIntegrationId, assetWithRegion)
			items.Keys = append(items.Keys, map[string]*dynamodb.AttributeValue{
				"AccountId": {
					S: aws.String(accountId),
				},
				"ATR": {
					S: aws.String(atr),
				},
			})

		}
	}
	BatchGetItems(&items, tableName)
}

func WriteAtrsToDynamo(accountId, stackId, awsIntegrationId, k8sIntegrationId, tableName string, assetTypesWithRegions []string, dynamoTTL int) {
	var items []*dynamodb.WriteRequest
	for _, assetWithRegion := range assetTypesWithRegions {
		var atr, providerType string
		if strings.HasPrefix(assetWithRegion, "aws") && len(awsIntegrationId) > 0 {
			atr = fmt.Sprintf("%s-%s", awsIntegrationId, assetWithRegion)
			providerType = "aws"

		} else if strings.HasPrefix(assetWithRegion, "kubernetes") {
			atr = fmt.Sprintf("%s-%s", k8sIntegrationId, assetWithRegion)
			providerType = "k8s"
		}
		currentTimeStamp := time.Now().Unix()
		expirationDate := goKitTypes.ToString(currentTimeStamp + int64(dynamoTTL) + 5)
		LastTriggered := goKitTypes.ToString(currentTimeStamp)
		itemToCreate :=  map[string]*dynamodb.AttributeValue{
			"AccountId": {
				S: aws.String(accountId),
			},
			"ATR": {
				S: aws.String(atr),
			},
			"ProviderType": {
				S: aws.String(providerType),
			},
			"ExpirationDate": {
				N: aws.String(expirationDate),
			},
			"LastTriggered": {
				N: aws.String(LastTriggered),
			},
		}

		items = append(items, &dynamodb.WriteRequest{PutRequest: &dynamodb.PutRequest{Item: itemToCreate}})
	}
	BatchPutItems(items, tableName)
}


func GetAtrsFromDynamo(accountId,  tableName string, atrs []string, client *goKitDynamo.Client ) {
	items := dynamodb.KeysAndAttributes{}

	for _, atr := range atrs {
		items.Keys = append(items.Keys,  map[string]*dynamodb.AttributeValue{
			"AccountId": {
				S: aws.String(accountId),
			},
			"ATR": {
				S: aws.String(atr),
			},
		})
	}
	batchItems, err := client.BatchGetItems(&items, tableName)

}

