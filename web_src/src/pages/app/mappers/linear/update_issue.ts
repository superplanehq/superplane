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
import { addTeamMetadata, buildIssueDetails, buildIssueSubtitle } from "./utils";
import type { LinearNodeMetadata, UpdateIssueConfiguration } from "./types";

export const updateIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return linearComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildIssueDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildIssueSubtitle(context.execution);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as LinearNodeMetadata | undefined;
  const configuration = node.configuration as UpdateIssueConfiguration | undefined;

  addTeamMetadata(metadata, nodeMetadata?.team, configuration?.team);

  if (configuration?.issue) {
    metadata.push({ icon: "hash", label: configuration.issue });
  }

  return metadata;
}
