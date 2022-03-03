package formatting

import (
	"encoding/json"
	"fmt"

	"github.com/marjamis/docket/lambda/reader/pkg/event"
)

func FormatEventData(evd *event.EventData) (output string) {
	var eventJsonFormattedString string
	switch evd.DetailType {
	case "ECS Container Instance State Change":
		var tempe event.ContainerInstanceStateChangeEvent
		err := json.Unmarshal([]byte(evd.EventJSON), &tempe)
		if err != nil {
			fmt.Println(err)
		}
		connection := "not connected"
		if tempe.AgentConnected {
			connection = "connected"
		}
		eventJsonFormattedString = "Status: " + tempe.Status + " and the agent is " + connection
	case "ECS Service Action":
		var tempe event.ServiceActionEvent
		err := json.Unmarshal([]byte(evd.EventJSON), &tempe)
		if err != nil {
			fmt.Println(err)
		}

		if tempe.EventType == "ERROR" {
			eventJsonFormattedString = "Error of \"" + tempe.EventName + "\" for reason: \"" + tempe.Reason + "\""
		} else if tempe.EventType == "INFO" && tempe.EventName == "SERVICE_STEADY_STATE" {
			eventJsonFormattedString = "Service entered the steady state"
		}
	case "AWS API Call via CloudTrail":
		return fmt.Sprintf("%s\t%s", evd.Time, evd.DetailType)
	default:
		eventJsonFormattedString = "Missing event details"
	}

	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", evd.Time, evd.DetailType, evd.Resources[0], eventJsonFormattedString, evd.ID)
}
