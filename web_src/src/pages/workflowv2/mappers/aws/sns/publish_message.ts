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

interface PublishMessageConfiguration {
  region?: string;
  topicArn?: string;
  format?: string;
}

interface PublishMessageData {
  messageId?: string;
  topicArn?: string;
}

export const publishMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext) {
    return buildSnsProps(context, buildMetadata(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as PublishMessageData | undefined;
    if (!result) {
      return {};
    }

    return {
      "Message ID": stringOrDash(result.messageId),
      "Topic ARN": stringOrDash(result.topicArn),
    };
  },

  subtitle(context: SubtitleContext): string {
    return buildSubtitle(context);
  },
};

function buildMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as PublishMessageConfiguration | undefined;
  const metadata: MetadataItem[] = [];

  const topicName = extractArnResourceName(configuration?.topicArn);
  if (topicName) {
    metadata.push({ icon: "hash", label: topicName });
  }

  const formatLabel = formatPublishMessageFormat(configuration?.format);
  if (formatLabel) {
    metadata.push({ icon: "message-square", label: formatLabel });
  }

  return metadata.slice(0, 2);
}

function formatPublishMessageFormat(format?: string): string | undefined {
  if (!format) {
    return undefined;
  }

  if (format.toLowerCase() === "json") {
    return "JSON";
  }

  if (format.toLowerCase() === "text") {
    return "Text";
  }

  return format;
}
