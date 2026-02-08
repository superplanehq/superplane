export interface SnsTriggerConfiguration {
  region?: string;
  topicArn?: string;
}

export interface SnsTriggerMetadata {
  region?: string;
  topicArn?: string;
  subscriptionId?: string;
}

export interface SnsTopicMessageEvent {
  account?: string;
  region?: string;
  source?: string;
  "detail-type"?: string;
  resources?: string[];
  detail?: {
    topicArn?: string;
    TopicArn?: string;
    subject?: string;
    Subject?: string;
    message?: string;
    Message?: string;
  };
}
