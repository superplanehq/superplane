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

interface DeleteTopicConfiguration {
  region?: string;
  topicArn?: string;
}

interface DeleteTopicData {
  topicArn?: string;
  deleted?: boolean;
}

export const deleteTopicMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return buildSnsProps(context, buildMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as DeleteTopicData | undefined;
    if (!result) {
      return {};
    }

    return {
      "Topic ARN": stringOrDash(result.topicArn),
      Deleted: stringOrDash(result.deleted),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildSubtitle(context);
  },
};

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as DeleteTopicConfiguration | undefined;
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
