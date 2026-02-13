import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
  NodeInfo,
} from "../../types";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildSnsProps, buildSubtitle, extractArnResourceName } from "./common";

interface GetSubscriptionConfiguration {
  region?: string;
  topicArn?: string;
  subscriptionArn?: string;
}

interface SubscriptionData {
  subscriptionArn?: string;
  topicArn?: string;
  protocol?: string;
  endpoint?: string;
  owner?: string;
  pendingConfirmation?: boolean;
  rawMessageDelivery?: boolean;
}

export const getSubscriptionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return buildSnsProps(context, buildMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as SubscriptionData | undefined;
    if (!result) {
      return {};
    }

    return {
      "Subscription ARN": stringOrDash(result.subscriptionArn),
      "Topic ARN": stringOrDash(result.topicArn),
      Protocol: stringOrDash(result.protocol),
      Endpoint: stringOrDash(result.endpoint),
      Owner: stringOrDash(result.owner),
      "Pending Confirmation": stringOrDash(result.pendingConfirmation),
      "Raw Message Delivery": stringOrDash(result.rawMessageDelivery),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildSubtitle(context);
  },
};

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as GetSubscriptionConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (configuration?.region) {
    items.push({ icon: "map", label: configuration.region });
  }

  const topicName = extractArnResourceName(configuration?.topicArn);
  if (topicName) {
    items.push({ icon: "hash", label: topicName });
  }

  const subscriptionArn = extractArnResourceName(configuration?.subscriptionArn);
  if (subscriptionArn) {
    items.push({ icon: "link", label: subscriptionArn });
  }

  return items;
}
