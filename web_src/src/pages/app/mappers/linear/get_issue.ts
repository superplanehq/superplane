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
import { buildIssueDetails, buildIssueSubtitle } from "./utils";
import type { GetIssueConfiguration } from "./types";

export const getIssueMapper: ComponentBaseMapper = {
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
  const configuration = node.configuration as GetIssueConfiguration | undefined;

  if (configuration?.issue) {
    metadata.push({ icon: "hash", label: configuration.issue });
  }

  return metadata;
}
