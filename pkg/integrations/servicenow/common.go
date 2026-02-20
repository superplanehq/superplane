package servicenow

import "fmt"

const (
	PayloadTypeIncident = "servicenow.incident"
)

type NodeMetadata struct {
	InstanceURL     string        `json:"instanceUrl,omitempty" mapstructure:"instanceUrl"`
	AssignmentGroup *ResourceInfo `json:"assignmentGroup,omitempty" mapstructure:"assignmentGroup"`
	AssignedTo      *ResourceInfo `json:"assignedTo,omitempty" mapstructure:"assignedTo"`
	Caller          *ResourceInfo `json:"caller,omitempty" mapstructure:"caller"`
	Incident        *ResourceInfo `json:"incident,omitempty" mapstructure:"incident"`
}

type ResourceInfo struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

type resourceSpec struct {
	AssignmentGroup string
	AssignedTo      string
	Caller          string
}

func resolveResourceMetadata(client *Client, spec resourceSpec) (NodeMetadata, error) {
	metadata := NodeMetadata{InstanceURL: client.InstanceURL}

	if spec.AssignmentGroup != "" {
		group, err := client.GetAssignmentGroup(spec.AssignmentGroup)
		if err != nil {
			return NodeMetadata{}, fmt.Errorf("error verifying assignment group: %w", err)
		}

		metadata.AssignmentGroup = &ResourceInfo{ID: group.SysID, Name: group.Name}
	}

	if spec.AssignedTo != "" {
		user, err := client.GetUser(spec.AssignedTo)
		if err != nil {
			return NodeMetadata{}, fmt.Errorf("error verifying assigned user: %w", err)
		}

		metadata.AssignedTo = &ResourceInfo{ID: user.SysID, Name: user.Name}
	}

	if spec.Caller != "" {
		user, err := client.GetUser(spec.Caller)
		if err != nil {
			return NodeMetadata{}, fmt.Errorf("error verifying caller: %w", err)
		}

		metadata.Caller = &ResourceInfo{ID: user.SysID, Name: user.Name}
	}

	return metadata, nil
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
