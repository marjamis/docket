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
	DetailType string    `json:"detail-type"`
	Source     string    `json:"source"`
	AccountID  string    `json:"account"`
	Time       time.Time `json:"time"`
	Region     string    `json:"region"`
	Resources  []string  `json:"resources"`
	EventJSON  string    `json:"event-json"`

	// This is just a catch-all for now
	CloudWatchEventPayload string `json:"cloudwatch-event-payload"`
	// Used to control when the event is expired in DDB
	EpochTTL int64 `json:"epoch-ttl"`
}

// ContainerInstanceStateChangeEvent is the important data for the event itself.
type ContainerInstanceStateChangeEvent struct {
	AgentConnected bool   `json:"agent-connected"`
	Status         string `json:"status"`
}

type containerState struct {
	ARN        string `json:"arn"`
	Name       string `json:"name"`
	LastStatus string `json:"last-status"`
}

//TaskStateChangeEvent is the important data for the event itself.
type TaskStateChangeEvent struct {
	LastStatus      string           `json:"last-status"`
	DesiredStatus   string           `json:"desired-status"`
	ContainerStates []containerState `json:"container-states"`
}

// ServiceActionEvent is the important data for the event itself.
type ServiceActionEvent struct {
	EventType           string   `json:"event_type"`
	EventName           string   `json:"event-name"`
	ClusterARN          string   `json:"cluster-arn"`
	Reason              string   `json:"reason,omitempty"`
	CapacityProvierARNs []string `json:"capacity-provider-arns,omitempty"`
}

// DeploymentEvent is the important data for the event itself.
type DeploymentEvent struct {
	EventType    string `json:"event-type"`
	EventName    string `json:"event-name"`
	DeploymentID string `json:"deployment-id"`
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

	customEntry := &EventData{
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

	t, err := dynamodbattribute.ConvertToMap(event)
	if err != nil {
		return nil, err
	}
	temp := t["detail"].M

	// Caters for the differences in the CWE that I may get for ECS and gets the details that I specifically want in the table
	switch event.DetailType {
	case "ECS Container Instance State Change":
		event := ContainerInstanceStateChangeEvent{
			AgentConnected: *temp["agentConnected"].BOOL,
			Status:         *temp["status"].S,
		}

		eventJSON, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}

		customEntry.EventJSON = string(eventJSON)
	case "ECS Service Action":
		event := ServiceActionEvent{
			EventType:  *temp["eventType"].S,
			EventName:  *temp["eventName"].S,
			ClusterARN: *temp["clusterArn"].S,
		}

		if i, ok := temp["reason"]; ok {
			event.Reason = *i.S
		}

		if i, ok := temp["capacityProviderArns"]; ok {
			for _, cap := range i.L {
				event.CapacityProvierARNs = append(event.CapacityProvierARNs, *cap.S)
			}
		}

		eventJSON, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}

		customEntry.EventJSON = string(eventJSON)

	case "ECS Deployment State Change":
		event := DeploymentEvent{
			EventType:    *temp["eventType"].S,
			EventName:    *temp["eventName"].S,
			Reason:       *temp["reason"].S,
			DeploymentID: *temp["deploymentId"].S,
		}

		eventJSON, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}

		customEntry.EventJSON = string(eventJSON)
	case "ECS Task State Change":
		event := TaskStateChangeEvent{
			LastStatus:    *temp["lastStatus"].S,
			DesiredStatus: *temp["desiredStatus"].S,
		}

		for _, s := range temp["containers"].L {
			c := s.M
			n := containerState{
				ARN:        *c["containerArn"].S,
				Name:       *c["name"].S,
				LastStatus: *c["lastStatus"].S,
			}
			event.ContainerStates = append(event.ContainerStates, n)
		}
	}

	// This converts the custom event struct data to a DDB item map
	entry, err := dynamodbattribute.ConvertToMap(*customEntry)
	if err != nil {
		return nil, err
	}

	return entry, nil
}
