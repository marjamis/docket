package cmd

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-xray-sdk-go/xray"

	log "github.com/sirupsen/logrus"
)

// EventData is the output that will be stored to DDB.
type EventData struct {
	ID         string    `json:"id"`
	DetailType string    `json:"detailType"`
	Source     string    `json:"source"`
	AccountID  string    `json:"account"`
	Time       time.Time `json:"time"`
	Region     string    `json:"region"`
	Resources  []string  `json:"resources"`
	EventJSON  string    `json:"eventJson"`

	// This is just a catch-all for now
	CloudWatchEventPayload string `json:"cloudwatchEventPayload"`
	// Used to control when the event is expired in DDB
	EpochTTL int64 `json:"epochTTL"`
}

// ContainerInstanceStateChangeEvent is the important data for the event itself.
type ContainerInstanceStateChangeEvent struct {
	AgentConnected bool   `json:"agentConnected"`
	Status         string `json:"status"`
}

type containerState struct {
	ARN        string `json:"containerArn"`
	Name       string `json:"name"`
	LastStatus string `json:"lastStatus"`
}

//TaskStateChangeEvent is the important data for the event itself.
type TaskStateChangeEvent struct {
	LastStatus    string           `json:"lastStatus"`
	DesiredStatus string           `json:"desiredStatus"`
	Containers    []containerState `json:"containers"`
}

// ServiceActionEvent is the important data for the event itself.
type ServiceActionEvent struct {
	EventType            string   `json:"eventType"`
	EventName            string   `json:"eventName"`
	ClusterARN           string   `json:"clusterArn"`
	Reason               string   `json:"reason,omitempty"`
	CapacityProviderARNs []string `json:"capacityProviderArns,omitempty"`
}

// DeploymentEvent is the important data for the event itself.
type DeploymentEvent struct {
	EventType    string `json:"eventType"`
	EventName    string `json:"eventName"`
	DeploymentID string `json:"deploymentId"`
	Reason       string `json:"reason"`
}

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
	// This is used to provide the full event in the table, as a backup in case this is needed later
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	ddb_item := &EventData{
		EpochTTL:               time.Now().AddDate(0, 0, 7).Unix(),
		CloudWatchEventPayload: string(payload),
		ID:                     event.ID,
		DetailType:             event.DetailType,
		Source:                 event.Source,
		AccountID:              event.AccountID,
		Time:                   event.Time,
		Region:                 event.Region,
		Resources:              event.Resources,
	}

	// Using this as a temporary method to get easy struct access to the data for the detail, which is a json.RawMessage
	t, err := dynamodbattribute.ConvertToMap(event)
	if err != nil {
		return nil, err
	}
	detail := t["detail"].M

	// Caters for the differences in the CWE that I may get for ECS and gets the details that I specifically want in the table
	switch event.DetailType {
	case "ECS Container Instance State Change":
		e := ContainerInstanceStateChangeEvent{
			AgentConnected: *detail["agentConnected"].BOOL,
			Status:         *detail["status"].S,
		}

		eventJSON, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}

		ddb_item.EventJSON = string(eventJSON)
	case "ECS Service Action":
		e := ServiceActionEvent{
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

		ddb_item.EventJSON = string(eventJSON)
	case "ECS Deployment State Change":
		e := DeploymentEvent{
			EventType:    *detail["eventType"].S,
			EventName:    *detail["eventName"].S,
			Reason:       *detail["reason"].S,
			DeploymentID: *detail["deploymentId"].S,
		}

		eventJSON, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}

		ddb_item.EventJSON = string(eventJSON)
	case "ECS Task State Change":
		e := TaskStateChangeEvent{
			LastStatus:    *detail["lastStatus"].S,
			DesiredStatus: *detail["desiredStatus"].S,
		}

		for _, s := range detail["containers"].L {
			c := s.M
			n := containerState{
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

		ddb_item.EventJSON = string(eventJSON)
	}

	// This converts the custom event struct data to a DDB item map
	entry, err := dynamodbattribute.ConvertToMap(*ddb_item)
	if err != nil {
		return nil, err
	}

	return entry, nil
}
