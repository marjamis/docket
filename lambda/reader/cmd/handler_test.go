package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/marjamis/docket/lambda/reader/pkg/event"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	fuzz "github.com/google/gofuzz"

	"github.com/stretchr/testify/assert"
)

func TestFormatDDBEntryFuzz(t *testing.T) {
	f := fuzz.New().NilChance(0).Funcs(
		func(e *events.CloudWatchEvent, c fuzz.Continue) {
			switch c.Intn(43) {
			case 0:
				e.DetailType = "ECS Container Instance State Change"
				detail := event.ContainerInstanceStateChangeEvent{}
				c.Fuzz(&detail)
				jsonDetail, err := json.Marshal(detail)
				if err != nil {
					fmt.Println(err)
				}
				e.Detail = jsonDetail
			case 1:
				e.DetailType = "ECS Service Action"
				detail := event.ServiceActionEvent{}
				c.Fuzz(&detail)
				jsonDetail, err := json.Marshal(detail)
				if err != nil {
					fmt.Println(err)
				}
				e.Detail = jsonDetail
			case 2:
				e.DetailType = "ECS Deployment State Change"
				detail := event.DeploymentEvent{}
				c.Fuzz(&detail)
				jsonDetail, err := json.Marshal(detail)
				if err != nil {
					fmt.Println(err)
				}
				e.Detail = jsonDetail
			case 3:
				e.DetailType = "ECS Task State Change"
				detail := event.TaskStateChangeEvent{}
				c.Fuzz(&detail)
				jsonDetail, err := json.Marshal(detail)
				if err != nil {
					fmt.Println(err)
				}
				e.Detail = jsonDetail
			}
		},
	)

	for i := 0; i < 10000; i++ {
		var object events.CloudWatchEvent
		f.Fuzz(&object)

		data, err := formatDDBEntry(object)
		if err != nil {
			fmt.Println(err)
		}

		t.Run("Checking Common Attributes", func(t *testing.T) {
			assert.Equal(t, object.ID, *data["id"].S)
			assert.Equal(t, object.DetailType, *data["detailType"].S)
			assert.Equal(t, object.Source, *data["source"].S)
			assert.Equal(t, object.AccountID, *data["account"].S)
			// RFC3339Nano is the time format used by DynamoDB Attribute
			assert.Equal(t, object.Time.Format(time.RFC3339Nano), *data["time"].S)
			assert.Equal(t, object.Region, *data["region"].S)

			var r []*dynamodb.AttributeValue
			for i := range object.Resources {
				r = append(r, &dynamodb.AttributeValue{S: &object.Resources[i]})
			}
			assert.Equal(t, r, data["resources"].L)

			epoch, err := strconv.Atoi(*data["epochTTL"].N)
			if err != nil {
				fmt.Println(err)
			}
			// TODO improve this check to be more "accurate"
			assert.Greater(t, int64(epoch), time.Now().Unix())
		})

		switch object.DetailType {
		case "ECS Container Instance State Change":
			respEventJSON := event.ContainerInstanceStateChangeEvent{}
			err = json.Unmarshal([]byte(*data["eventJson"].S), &respEventJSON)
			if err != nil {
				fmt.Println(err)
			}

			var detail event.ContainerInstanceStateChangeEvent
			err := json.Unmarshal(object.Detail, &detail)
			if err != nil {
				fmt.Println(err)
			}

			t.Run("Checking ECS Container Instance State Change Event Data", func(t *testing.T) {
				assert.Equal(t, detail.AgentConnected, respEventJSON.AgentConnected)
				assert.Equal(t, detail.Status, respEventJSON.Status)
			})
		case "ECS Service Action":
			respEventJSON := event.ServiceActionEvent{}
			err = json.Unmarshal([]byte(*data["eventJson"].S), &respEventJSON)
			if err != nil {
				fmt.Println(err)
			}

			var detail event.ServiceActionEvent
			err := json.Unmarshal(object.Detail, &detail)
			if err != nil {
				fmt.Println(err)
			}

			t.Run("Checking ECS Service Action Event Data", func(t *testing.T) {
				assert.Equal(t, detail.EventType, respEventJSON.EventType)
				assert.Equal(t, detail.EventName, respEventJSON.EventName)
				assert.Equal(t, detail.ClusterARN, respEventJSON.ClusterARN)
				// TODO this is an imperfect test as sometimes the json shouldn't have either of these keys
				assert.Equal(t, detail.Reason, respEventJSON.Reason)
				assert.Equal(t, detail.CapacityProviderARNs, respEventJSON.CapacityProviderARNs)
			})
		case "ECS Deployment State Change":
			respEventJSON := event.DeploymentEvent{}
			err = json.Unmarshal([]byte(*data["eventJson"].S), &respEventJSON)
			if err != nil {
				fmt.Println(err)
			}

			var detail event.DeploymentEvent
			err := json.Unmarshal(object.Detail, &detail)
			if err != nil {
				fmt.Println(err)
			}

			t.Run("Checking ECS Deployment Event Data", func(t *testing.T) {
				assert.Equal(t, detail.EventType, respEventJSON.EventType)
				assert.Equal(t, detail.EventName, respEventJSON.EventName)
				assert.Equal(t, detail.Reason, respEventJSON.Reason)
				assert.Equal(t, detail.DeploymentID, respEventJSON.DeploymentID)
			})
		case "ECS Task State Change":
			respEventJSON := event.TaskStateChangeEvent{}
			err = json.Unmarshal([]byte(*data["eventJson"].S), &respEventJSON)
			if err != nil {
				fmt.Println(err)
			}

			var detail event.TaskStateChangeEvent
			err := json.Unmarshal(object.Detail, &detail)
			if err != nil {
				fmt.Println(err)
			}

			t.Run("Checking ECS Task State Event Data", func(t *testing.T) {
				assert.Equal(t, detail.LastStatus, respEventJSON.LastStatus)
				assert.Equal(t, detail.DesiredStatus, respEventJSON.DesiredStatus)
				assert.Equal(t, detail.Containers, respEventJSON.Containers)
			})
		}
	}
}
