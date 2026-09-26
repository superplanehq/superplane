package messages

import "encoding/json"

const SupportFeedbackRequestedRoutingKey = "support-feedback-requested"

type SupportFeedbackRequestedMessage struct {
	Category         string                     `json:"category"`
	Details          string                     `json:"details"`
	UserName         string                     `json:"user_name"`
	UserEmail        string                     `json:"user_email"`
	OrganizationID   string                     `json:"organization_id"`
	OrganizationName string                     `json:"organization_name"`
	PagePath         string                     `json:"page_path,omitempty"`
	Attachment       *SupportFeedbackAttachment `json:"attachment,omitempty"`
	EmailDelivered   bool                       `json:"email_delivered,omitempty"`
	DiscordDelivered bool                       `json:"discord_delivered,omitempty"`
}

type SupportFeedbackAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	BlobKey     string `json:"blob_key"`
}

func (m SupportFeedbackRequestedMessage) Publish() error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return Publish(CanvasExchange, SupportFeedbackRequestedRoutingKey, body)
}
