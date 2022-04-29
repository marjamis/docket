package cmd

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/marjamis/docket/lambda/reader/pkg/event"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	log "github.com/sirupsen/logrus"
)

// HandleRequest is used as the lambda function's handler to start all processing, essentially the workflow of the function's execution.
func HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	log.Debug("Initial payload: ", event)

	item, err := formatDDBEntry(event)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return ddbPutItem(item)
}

// dbbPutItem will take a DynamoDB formatted map as the input item and put this item in a DynamoDB table.
func ddbPutItem(item map[string]*dynamodb.AttributeValue) error {
	// TODO move this outside of the handler in a go way
	svc := dynamodb.New(session.New(&aws.Config{}))

	_, err := svc.PutItem(&dynamodb.PutItemInput{
		Item: item,
		// TODO Perhaps store this in the same place as when the client is outside of the handler
		TableName: aws.String(os.Getenv("DDB_TABLE")),
	})

	return err
}

// formatDDBEntry will take an ECS  CloudWatch Event and convert it to a DynamoDB approiate item with some customisations for
// what is stored as a field in the DDB table.
func formatDDBEntry(cwe events.CloudWatchEvent) (map[string]*dynamodb.AttributeValue, error) {
	// This is used to provide the full event in the table, as a backup in case this is needed later
	payload, err := json.Marshal(cwe)
	if err != nil {
		return nil, err
	}

	ddbItem := &event.EventData{
		EpochTTL:               time.Now().AddDate(0, 0, 7).Unix(),
		CloudWatchEventPayload: string(payload),
		ID:                     cwe.ID,
		DetailType:             cwe.DetailType,
		Source:                 cwe.Source,
		AccountID:              cwe.AccountID,
		Time:                   cwe.Time,
		Region:                 cwe.Region,
		Resources:              cwe.Resources,
	}

	// Using this as a temporary method to get easy struct access to the data for the detail, which is a json.RawMessage
	t, err := dynamodbattribute.ConvertToMap(cwe)
	if err != nil {
		return nil, err
	}
	detail := t["detail"].M

	// Caters for the differences in the CWE that I may get for ECS and gets the details that I specifically want in the table
	switch cwe.DetailType {
	case "ECS Container Instance State Change":
		e := event.ContainerInstanceStateChangeEvent{
			AgentConnected: *detail["agentConnected"].BOOL,
			Status:         *detail["status"].S,
		}

		eventJSON, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}

		ddbItem.EventJSON = string(eventJSON)
	case "ECS Service Action":
		e := event.ServiceActionEvent{
			EventType:  *detail["eventType"].S,
			EventName:  *detail["eventName"].S,
			ClusterARN: *detail["clusterArn"].S,
		}

		if i, ok := detail["reason"]; ok {
			e.Reason = *i.S
		}

		if i, ok := detail["capacityProviderArns"]; ok {
			for _, cap := range i.L {
				e.CapacityProviderARNs = append(e.CapacityProviderARNs, *cap.S)
			}
		}

		eventJSON, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}

		ddbItem.EventJSON = string(eventJSON)
	case "ECS Deployment State Change":
		e := event.DeploymentEvent{
			EventType:    *detail["eventType"].S,
			EventName:    *detail["eventName"].S,
			Reason:       *detail["reason"].S,
			DeploymentID: *detail["deploymentId"].S,
		}

		eventJSON, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}

		ddbItem.EventJSON = string(eventJSON)
	case "ECS Task State Change":
		e := event.TaskStateChangeEvent{
			LastStatus:    *detail["lastStatus"].S,
			DesiredStatus: *detail["desiredStatus"].S,
		}

		for _, s := range detail["containers"].L {
			c := s.M
			n := event.ContainerState{
				ARN:        *c["containerArn"].S,
				Name:       *c["name"].S,
				LastStatus: *c["lastStatus"].S,
			}
			e.Containers = append(e.Containers, n)
		}

		eventJSON, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}

		ddbItem.EventJSON = string(eventJSON)
	}

	// This converts the custom event struct data to a DDB item map
	entry, err := dynamodbattribute.ConvertToMap(*ddbItem)
	if err != nil {
		return nil, err
	}

	return entry, nil
}
