import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { linearComponentBaseProps } from "./base";
import { buildAttachmentDeletionDetails } from "./utils";
import type { DeleteAttachmentConfiguration } from "./types";

export const deleteAttachmentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return linearComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildAttachmentDeletionDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const { execution } = context;
    return execution.createdAt ? renderTimeAgo(new Date(execution.createdAt)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as DeleteAttachmentConfiguration | undefined;

  if (configuration?.issue) {
    metadata.push({ icon: "hash", label: configuration.issue });
  }

  if (configuration?.attachment) {
    metadata.push({ icon: "link", label: configuration.attachment });
  }

  return metadata;
}
