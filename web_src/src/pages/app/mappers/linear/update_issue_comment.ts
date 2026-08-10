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
import { buildCommentSubtitle, buildCommentUpdateDetails } from "./utils";
import type { UpdateIssueCommentConfiguration } from "./types";

export const updateIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return linearComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildCommentUpdateDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildCommentSubtitle(context.execution);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as UpdateIssueCommentConfiguration | undefined;

  if (configuration?.issue) {
    metadata.push({ icon: "hash", label: configuration.issue });
  }

  return metadata;
}
