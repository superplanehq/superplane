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
import type { CreateIssueConfiguration, LinearNodeMetadata } from "./types";

export const createIssueMapper: ComponentBaseMapper = {
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
  const configuration = node.configuration as CreateIssueConfiguration | undefined;

  addTeamMetadata(metadata, nodeMetadata?.team, configuration?.team);

  const priority = priorityLabel(configuration?.priority);
  if (priority) {
    metadata.push({ icon: "flag", label: priority });
  }

  return metadata;
}

/** Linear encodes priority as 0-4; the picker stores the number as a string. */
const PRIORITY_LABELS: Record<string, string> = {
  "0": "No priority",
  "1": "Urgent",
  "2": "High",
  "3": "Medium",
  "4": "Low",
};

function priorityLabel(priority: string | undefined): string | undefined {
  if (!priority) return undefined;
  return PRIORITY_LABELS[priority];
}
