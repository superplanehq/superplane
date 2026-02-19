package servicenow

const (
	PayloadTypeIncident  = "servicenow.incident"
	PayloadTypeIncidents = "servicenow.incidents.list"
)

type NodeMetadata struct {
	InstanceURL     string        `json:"instanceUrl,omitempty" mapstructure:"instanceUrl"`
	AssignmentGroup *ResourceInfo `json:"assignmentGroup,omitempty" mapstructure:"assignmentGroup"`
	AssignedTo      *ResourceInfo `json:"assignedTo,omitempty" mapstructure:"assignedTo"`
	Caller          *ResourceInfo `json:"caller,omitempty" mapstructure:"caller"`
}

type ResourceInfo struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

type IncidentRecord struct {
	SysID            string `json:"sys_id"`
	Number           string `json:"number"`
	ShortDescription string `json:"short_description"`
	State            string `json:"state"`
	Urgency          string `json:"urgency"`
	Impact           string `json:"impact"`
	Priority         string `json:"priority"`
	Category         string `json:"category"`
	Subcategory      string `json:"subcategory"`
	SysCreatedOn     string `json:"sys_created_on"`
	SysUpdatedOn     string `json:"sys_updated_on"`
}
