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
import { buildCommentDetails, buildCommentSubtitle } from "./utils";
import type { AddIssueCommentConfiguration } from "./types";

export const addIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return linearComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildCommentDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildCommentSubtitle(context.execution);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as AddIssueCommentConfiguration | undefined;

  if (configuration?.issue) {
    metadata.push({ icon: "hash", label: configuration.issue });
  }

  return metadata;
}
