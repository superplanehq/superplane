export interface SnsTriggerConfiguration {
  region?: string;
  topicArn?: string;
}

export interface SnsTriggerMetadata {
  region?: string;
  topicArn?: string;
  subscriptionArn?: string;
  webhookUrl?: string;
}

export interface SnsTopicMessageEvent {
  type?: string;
  messageId?: string;
  topicArn?: string;
  subject?: string;
  message?: string;
  timestamp?: string;
  account?: string;
  region?: string;
  source?: string;
  "detail-type"?: string;
  resources?: string[];
  detail?: {
    messageId?: string;
    topicArn?: string;
    TopicArn?: string;
    subject?: string;
    Subject?: string;
    message?: string;
    Message?: string;
  };
}
