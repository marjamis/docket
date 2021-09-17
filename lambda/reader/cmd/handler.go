package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-xray-sdk/xray"

	log "github.com/sirupsen/logrus/"
)

// HandleRequest is used as the lambda function's handler to start all processing, essentially the workflow of the function's execution.
func HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	log.Debug("Main payload: %+v\n", event)

	item, err := formatDDBEntry(event)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return ddbPutItem(item)
}

// dbbPutItem will take a DynamoDB formatting map as the input item and put this item in a DynamoDB table.
func ddbPutItem(item map[string]*dynamodb.AttributeValue) error {
	// TODO move this outside of the handler in a go way
	svc := dynamodb.New(session.New(&aws.Config{}))
	xray.AWS(svc.Client)

	_, err := svc.PutItem(&dynamodb.PutItemInput{
		Item: item,
		// TODO Perhaps store this in the same place as when the client is outside of the handler
		TableName: aws.String(os.Getenv("DDB_TABLE")),
	})

	return err
}

// formatDDBEntry will take an ECS  CloudWatch Event and convert it to a DynamoDB approiate item with some customisations for
// what is stored as a field in the DDB table.
func formatDDBEntry(event events.CloudWatchEvent) (map[string]*dynamodb.AttributeValue, error) {
	// This will automatically convert the entire CWE to a DDB item map
	entry, err := dynamodbattribute.ConvertToMap(event)
	if err != nil {
		return nil, err
	}

	// This creates the setting for the TTL of the item in DDB
	entry["epochTTL"] = &dynamodb.AttributeValue{
		// The current expiration is 7 days after now
		N: aws.String(strconv.FormatInt(time.Now().AddDate(0, 0, 7).Unix(), 10)),
	}

	// This is used to provide the full event in the table, as a backup in case this is needed later
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	//TODO can I collapse to remove unwanted spaces - this doesn't work but fix later
	var compactPayload bytes.Buffer
	json.Compact(&compactPayload, payload)
	entry["cloudwatch-event-payload"] = &dynamodb.AttributeValue{
		S: aws.String(compactPayload.String()),
	}

	// As the conversion is already dont I simply use the converted data to pull what I want out that will be in specific fields of the table
	detail := entry["detail"].M

	// Caters for the differences in the CWE that I may get for ECS and gets the details that I specifically want in the table
	switch event.DetailType {
	case "ECS Container Instance State Change":
		entry["agentConnected"] = detail["agentConnected"]
		entry["status"] = detail["status"]
	case "ECS Service Action":
		for k, v := range detail {
			entry[k] = v
		}
	case "ECS Deployment State Change":
		for k, v := range detail {
			entry[k] = v
		}
	case "ECS Task State Change":
		entry["availabilityZone"] = detail["availabilityZone"]
		entry["containers"] = detail["containers"]
	}

	// As this detail section isn't needed in the final output, as it's backed up elsewhere, this key is deleted
	delete(entry, "detail")

	return entry, nil
}
