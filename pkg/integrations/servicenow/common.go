package servicenow

const PayloadTypeIncident = "servicenow.incident"

type NodeMetadata struct {
	WebhookURL      string        `json:"webhookUrl,omitempty" mapstructure:"webhookUrl"`
	InstanceURL     string        `json:"instanceUrl,omitempty" mapstructure:"instanceUrl"`
	AssignmentGroup *ResourceInfo `json:"assignmentGroup,omitempty" mapstructure:"assignmentGroup"`
	AssignedTo      *ResourceInfo `json:"assignedTo,omitempty" mapstructure:"assignedTo"`
	Caller          *ResourceInfo `json:"caller,omitempty" mapstructure:"caller"`
}

type ResourceInfo struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}
