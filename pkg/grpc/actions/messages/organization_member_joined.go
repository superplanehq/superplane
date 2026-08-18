package messages

import "encoding/json"

const OrganizationMemberJoinedRoutingKey = "organization-member-joined"

type OrganizationMemberJoinedMessage struct {
	ToEmail          string `json:"to_email"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	MemberEmail      string `json:"member_email"`
	MemberName       string `json:"member_name,omitempty"`
}

func (m OrganizationMemberJoinedMessage) Publish() error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return Publish(CanvasExchange, OrganizationMemberJoinedRoutingKey, body)
}
