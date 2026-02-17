package common

const (
	WebhookTypeSNS = "sns"
)

type WebhookConfiguration struct {
	Region string                   `json:"region"`
	Type   string                   `json:"type"`
	SNS    *SNSWebhookConfiguration `json:"sns"`
}

type SNSWebhookConfiguration struct {
	TopicArn string `json:"topicArn"`
}

type SNSWebhookMetadata struct {
	SubscriptionArn string `json:"subscriptionArn"`
}
