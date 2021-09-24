package event

import "time"

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

// ContainerState contains the details of a container.
type ContainerState struct {
	ARN        string `json:"containerArn"`
	Name       string `json:"name"`
	LastStatus string `json:"lastStatus"`
}

//TaskStateChangeEvent is the important data for the event itself.
type TaskStateChangeEvent struct {
	LastStatus    string           `json:"lastStatus"`
	DesiredStatus string           `json:"desiredStatus"`
	Containers    []ContainerState `json:"containers"`
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
