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

interface GetTopicConfiguration {
  region?: string;
  topicArn?: string;
}

interface TopicData {
  topicArn?: string;
  name?: string;
  displayName?: string;
  owner?: string;
  fifoTopic?: boolean;
  contentBasedDeduplication?: boolean;
}

export const getTopicMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return buildSnsProps(context, buildMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as TopicData | undefined;
    if (!result) {
      return {};
    }

    return {
      "Topic ARN": stringOrDash(result.topicArn),
      Name: stringOrDash(result.name),
      "Display Name": stringOrDash(result.displayName),
      Owner: stringOrDash(result.owner),
      "FIFO Topic": stringOrDash(result.fifoTopic),
      "Content-based Deduplication": stringOrDash(result.contentBasedDeduplication),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildSubtitle(context);
  },
};

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as GetTopicConfiguration | undefined;
  const topicName = extractArnResourceName(configuration?.topicArn);
  const items: MetadataItem[] = [];

  if (configuration?.region) {
    items.push({ icon: "map", label: configuration.region });
  }

  if (topicName) {
    items.push({ icon: "hash", label: topicName });
  }

  return items;
}
