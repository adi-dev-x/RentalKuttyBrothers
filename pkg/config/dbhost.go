package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// dbServerID is the server_infra item that holds the database IP.
const dbServerID = "db-server"

// serverItem is one row of the server_infra table, as written by the
// server-infra lambda.
type serverItem struct {
	ID          string `dynamodbav:"id"`
	InstanceID  string `dynamodbav:"instanceID"`
	IP          string `dynamodbav:"ip"`
	RequestTime string `dynamodbav:"requestTime"`
	Status      bool   `dynamodbav:"status"`
	Threshold   int    `dynamodbav:"threshold"`
}

// ResolveDBHost sets PGHost to the IP of the db-server item in DynamoDB.
// If the lookup fails, the PG_HOST from the env is kept when present.
func ResolveDBHost(conf *Config) error {

	ip, err := fetchDBServerIP(*conf)

	if err != nil {
		if conf.PGHost == "" {
			return fmt.Errorf("db host: %w", err)
		}

		log.Printf("db host: %v, using PG_HOST=%s", err, conf.PGHost)

		return nil
	}

	log.Printf("db host: using %s from dynamodb", ip)
	conf.PGHost = ip

	return nil
}

func fetchDBServerIP(conf Config) (string, error) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	region := conf.DY_AWS_REGION
	if region == "" {
		region = "eu-north-1"
	}

	tableName := "server_infra"
	if value := os.Getenv("DYNAMODB_TABLE"); value != "" {
		tableName = value
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}

	if conf.AWS_ACCKEY != "" && conf.AWS_SECKEY != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(conf.AWS_ACCKEY, conf.AWS_SECKEY, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("load aws config: %w", err)
	}

	result, err := dynamodb.NewFromConfig(awsCfg).GetItem(
		ctx,
		&dynamodb.GetItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{
					Value: dbServerID,
				},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("get %s from %s: %w", dbServerID, tableName, err)
	}

	if len(result.Item) == 0 {
		return "", fmt.Errorf("%s not found in %s", dbServerID, tableName)
	}

	var item serverItem

	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return "", fmt.Errorf("read %s: %w", dbServerID, err)
	}

	if item.IP == "" {
		return "", fmt.Errorf("%s has no ip", dbServerID)
	}

	return item.IP, nil
}
