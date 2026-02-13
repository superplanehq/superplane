package sns

// Topic models an AWS SNS topic payload returned by SNS API operations.
type Topic struct {
	TopicArn                  string            `json:"topicArn" mapstructure:"topicArn"`
	Name                      string            `json:"name" mapstructure:"name"`
	DisplayName               string            `json:"displayName,omitempty" mapstructure:"displayName"`
	Owner                     string            `json:"owner,omitempty" mapstructure:"owner"`
	KmsMasterKeyID            string            `json:"kmsMasterKeyId,omitempty" mapstructure:"kmsMasterKeyId"`
	FifoTopic                 bool              `json:"fifoTopic" mapstructure:"fifoTopic"`
	ContentBasedDeduplication bool              `json:"contentBasedDeduplication" mapstructure:"contentBasedDeduplication"`
	Attributes                map[string]string `json:"attributes,omitempty" mapstructure:"attributes"`
}

// Subscription models an AWS SNS subscription payload returned by SNS API operations.
type Subscription struct {
	SubscriptionArn     string            `json:"subscriptionArn" mapstructure:"subscriptionArn"`
	TopicArn            string            `json:"topicArn,omitempty" mapstructure:"topicArn"`
	Protocol            string            `json:"protocol,omitempty" mapstructure:"protocol"`
	Endpoint            string            `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Owner               string            `json:"owner,omitempty" mapstructure:"owner"`
	PendingConfirmation bool              `json:"pendingConfirmation" mapstructure:"pendingConfirmation"`
	RawMessageDelivery  bool              `json:"rawMessageDelivery" mapstructure:"rawMessageDelivery"`
	Attributes          map[string]string `json:"attributes,omitempty" mapstructure:"attributes"`
}

// PublishResult models the response emitted after publishing an SNS message.
type PublishResult struct {
	MessageID      string `json:"messageId" mapstructure:"messageId"`
	SequenceNumber string `json:"sequenceNumber,omitempty" mapstructure:"sequenceNumber"`
	TopicArn       string `json:"topicArn" mapstructure:"topicArn"`
}

// GetTopicConfiguration contains user-provided settings for GetTopic.
type GetTopicConfiguration struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

// GetSubscriptionConfiguration contains user-provided settings for GetSubscription.
type GetSubscriptionConfiguration struct {
	Region          string `json:"region" mapstructure:"region"`
	SubscriptionArn string `json:"subscriptionArn" mapstructure:"subscriptionArn"`
}

// CreateTopicConfiguration contains user-provided settings for CreateTopic.
type CreateTopicConfiguration struct {
	Region     string         `json:"region" mapstructure:"region"`
	Name       string         `json:"name" mapstructure:"name"`
	Attributes map[string]any `json:"attributes" mapstructure:"attributes"`
}

// DeleteTopicConfiguration contains user-provided settings for DeleteTopic.
type DeleteTopicConfiguration struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

// PublishMessageConfiguration contains user-provided settings for PublishMessage.
type PublishMessageConfiguration struct {
	Region            string         `json:"region" mapstructure:"region"`
	TopicArn          string         `json:"topicArn" mapstructure:"topicArn"`
	Message           string         `json:"message" mapstructure:"message"`
	Subject           string         `json:"subject" mapstructure:"subject"`
	MessageAttributes map[string]any `json:"messageAttributes" mapstructure:"messageAttributes"`
}

// OnTopicMessageConfiguration contains user-provided settings for OnTopicMessage.
type OnTopicMessageConfiguration struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

// OnTopicMessageMetadata stores trigger setup state for OnTopicMessage.
type OnTopicMessageMetadata struct {
	Region          string `json:"region" mapstructure:"region"`
	TopicArn        string `json:"topicArn" mapstructure:"topicArn"`
	WebhookURL      string `json:"webhookUrl" mapstructure:"webhookUrl"`
	SubscriptionArn string `json:"subscriptionArn" mapstructure:"subscriptionArn"`
}

// PublishMessageParameters defines the arguments for a publish operation.
type PublishMessageParameters struct {
	TopicArn          string
	Message           string
	Subject           string
	MessageAttributes map[string]string
}

// SubscribeParameters defines the arguments for a subscribe operation.
type SubscribeParameters struct {
	TopicArn              string
	Protocol              string
	Endpoint              string
	Attributes            map[string]string
	ReturnSubscriptionARN bool
}

type snsWebhookMessage struct {
	Type              string                         `json:"Type"`
	MessageID         string                         `json:"MessageId"`
	TopicArn          string                         `json:"TopicArn"`
	Subject           string                         `json:"Subject"`
	Message           string                         `json:"Message"`
	Timestamp         string                         `json:"Timestamp"`
	SignatureVersion  string                         `json:"SignatureVersion"`
	Signature         string                         `json:"Signature"`
	SigningCertURL    string                         `json:"SigningCertURL"`
	UnsubscribeURL    string                         `json:"UnsubscribeURL"`
	SubscribeURL      string                         `json:"SubscribeURL"`
	Token             string                         `json:"Token"`
	MessageAttributes map[string]snsMessageAttribute `json:"MessageAttributes"`
}

type snsMessageAttribute struct {
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

type attributeEntry struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type getTopicAttributesResponse struct {
	Entries []attributeEntry `xml:"GetTopicAttributesResult>Attributes>entry"`
}

type createTopicResponse struct {
	TopicArn string `xml:"CreateTopicResult>TopicArn"`
}

type publishResponse struct {
	MessageID      string `xml:"PublishResult>MessageId"`
	SequenceNumber string `xml:"PublishResult>SequenceNumber"`
}

type getSubscriptionAttributesResponse struct {
	Entries []attributeEntry `xml:"GetSubscriptionAttributesResult>Attributes>entry"`
}

type subscribeResponse struct {
	SubscriptionArn string `xml:"SubscribeResult>SubscriptionArn"`
}

type listTopicMember struct {
	TopicArn string `xml:"TopicArn"`
}

type listTopicsResponse struct {
	Topics    []listTopicMember `xml:"ListTopicsResult>Topics>member"`
	NextToken string            `xml:"ListTopicsResult>NextToken"`
}

type listSubscriptionMember struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
	TopicArn        string `xml:"TopicArn"`
	Protocol        string `xml:"Protocol"`
	Endpoint        string `xml:"Endpoint"`
	Owner           string `xml:"Owner"`
}

type listSubscriptionsResponse struct {
	Subscriptions      []listSubscriptionMember `xml:"ListSubscriptionsResult>Subscriptions>member"`
	SubscriptionsTopic []listSubscriptionMember `xml:"ListSubscriptionsByTopicResult>Subscriptions>member"`
	NextToken          string                   `xml:"ListSubscriptionsResult>NextToken"`
	NextTokenTopic     string                   `xml:"ListSubscriptionsByTopicResult>NextToken"`
}

type snsErrorDetail struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type snsErrorPayload struct {
	Error snsErrorDetail `xml:"Error"`
}
