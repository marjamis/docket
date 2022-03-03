package formatting

import (
	"testing"
	"time"

	"github.com/marjamis/docket/lambda/reader/pkg/event"

	"github.com/stretchr/testify/assert"
)

func TestEventDataTableOutput(t *testing.T) {
	tests := []struct {
		input    *event.EventData
		expected string
	}{
		{
			&event.EventData{
				ID:         "123",
				DetailType: "ECS Container Instance State Change",
				Source:     "Some source",
				AccountID:  "358913853",
				Time:       time.Now(),
				Region:     "us-west-2",
				Resources: []string{
					"container_instance#1",
				},
				EventJSON:              "{\"agentConnected\": true, \"status\": \"ACTIVE\"}",
				CloudWatchEventPayload: "{\"a\": 2}",
				EpochTTL:               12351356231523,
			},
			"time\tECS Container Instance State Change\tcontainer_instance#1\tStatus: ACTIVE and the agent is connected",
		},
		{
			&event.EventData{
				ID:         "123",
				DetailType: "ECS Service Action",
				Source:     "Some source",
				AccountID:  "358913853",
				Time:       time.Now(),
				Region:     "us-west-2",
				Resources: []string{
					"service#1",
				},
				EventJSON:              "{\"eventType\": \"ERROR\", \"eventName\": \"SERVICE_TASK_PLACEMENT_FAILURE\", \"Reason\": \"RESOURCE:FARGATE\"}",
				CloudWatchEventPayload: "{\"a\": 2}",
				EpochTTL:               12351356231523,
			},
			"time\tECS Service Action\tservice#1\tError of \"SERVICE_TASK_PLACEMENT_FAILURE\" for reason: \"RESOURCE:FARGATE\"",
		},
	}

	for _, v := range tests {
		assert.Equal(t, v.expected, FormatEventData(v.input))
	}
}
