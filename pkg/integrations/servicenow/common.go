package servicenow

const PayloadTypeIncident = "servicenow.incident"

type NodeMetadata struct {
	WebhookURL      string        `json:"webhookUrl,omitempty"`
	InstanceURL     string        `json:"instanceUrl,omitempty"`
	AssignmentGroup *ResourceInfo `json:"assignmentGroup,omitempty"`
	AssignedTo      *ResourceInfo `json:"assignedTo,omitempty"`
	Caller          *ResourceInfo `json:"caller,omitempty"`
}

type ResourceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
