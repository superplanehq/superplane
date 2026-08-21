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
import { linearComponentBaseProps } from "./base";
import { buildAttachmentDetails, buildAttachmentSubtitle } from "./utils";
import type { CreateAttachmentConfiguration } from "./types";

export const createAttachmentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return linearComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildAttachmentDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildAttachmentSubtitle(context.execution);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreateAttachmentConfiguration | undefined;

  if (configuration?.issue) {
    metadata.push({ icon: "hash", label: configuration.issue });
  }

  if (configuration?.title) {
    metadata.push({ icon: "link", label: configuration.title });
  }

  return metadata;
}
