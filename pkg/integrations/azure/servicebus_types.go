package azure

// armServiceBusNamespace represents a Service Bus namespace from the ARM API.
type armServiceBusNamespace struct {
	ID         string                        `json:"id"`
	Name       string                        `json:"name"`
	Location   string                        `json:"location"`
	Properties armServiceBusNamespaceProps   `json:"properties"`
	SKU        armServiceBusSKU              `json:"sku"`
}

type armServiceBusNamespaceProps struct {
	ServiceBusEndpoint string `json:"serviceBusEndpoint"`
	Status             string `json:"status"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type armServiceBusSKU struct {
	Name string `json:"name"` // Basic, Standard, Premium
	Tier string `json:"tier"`
}

// armServiceBusQueue represents a Service Bus queue from the ARM API.
type armServiceBusQueue struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Properties armServiceBusQueueProps `json:"properties"`
}

type armServiceBusQueueProps struct {
	MessageCount                    int64  `json:"messageCount"`
	SizeInBytes                     int64  `json:"sizeInBytes"`
	MaxSizeInMegabytes              int    `json:"maxSizeInMegabytes"`
	LockDuration                    string `json:"lockDuration"`
	MaxDeliveryCount                int    `json:"maxDeliveryCount"`
	RequiresDuplicateDetection      bool   `json:"requiresDuplicateDetection"`
	RequiresSession                 bool   `json:"requiresSession"`
	DefaultMessageTimeToLive        string `json:"defaultMessageTimeToLive"`
	DeadLetteringOnMessageExpiration bool  `json:"deadLetteringOnMessageExpiration"`
	EnableBatchedOperations         bool   `json:"enableBatchedOperations"`
	Status                          string `json:"status"`
	CreatedAt                       string `json:"createdAt"`
	UpdatedAt                       string `json:"updatedAt"`
	AccessedAt                      string `json:"accessedAt"`
	CountDetails                    armServiceBusMessageCountDetails `json:"countDetails"`
}

type armServiceBusMessageCountDetails struct {
	ActiveMessageCount             int64 `json:"activeMessageCount"`
	DeadLetterMessageCount         int64 `json:"deadLetterMessageCount"`
	ScheduledMessageCount          int64 `json:"scheduledMessageCount"`
	TransferMessageCount           int64 `json:"transferMessageCount"`
	TransferDeadLetterMessageCount int64 `json:"transferDeadLetterMessageCount"`
}

// armServiceBusTopic represents a Service Bus topic from the ARM API.
type armServiceBusTopic struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Properties armServiceBusTopicProps `json:"properties"`
}

type armServiceBusTopicProps struct {
	SizeInBytes                     int64  `json:"sizeInBytes"`
	MaxSizeInMegabytes              int    `json:"maxSizeInMegabytes"`
	DefaultMessageTimeToLive        string `json:"defaultMessageTimeToLive"`
	RequiresDuplicateDetection      bool   `json:"requiresDuplicateDetection"`
	EnableBatchedOperations         bool   `json:"enableBatchedOperations"`
	EnablePartitioning              bool   `json:"enablePartitioning"`
	SupportOrdering                 bool   `json:"supportOrdering"`
	Status                          string `json:"status"`
	CreatedAt                       string `json:"createdAt"`
	UpdatedAt                       string `json:"updatedAt"`
	AccessedAt                      string `json:"accessedAt"`
	SubscriptionCount               int    `json:"subscriptionCount"`
	CountDetails                    armServiceBusMessageCountDetails `json:"countDetails"`
}
