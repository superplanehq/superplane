import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  MAX_METADATA_ITEMS,
  prometheusSubtitle,
  type WorkspaceOutput,
  workspaceExecutionDetails,
} from "./common";

interface CreateWorkspaceConfiguration {
  region?: string;
  alias?: string;
  kmsKeyArn?: string;
}

export const createWorkspaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, createWorkspaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<WorkspaceOutput>(context.execution.outputs);
    const config = context.node.configuration as CreateWorkspaceConfiguration | undefined;
    return workspaceExecutionDetails(data?.workspace, context.execution, "Created At", config?.alias, "created");
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

function createWorkspaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as CreateWorkspaceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.alias) {
    items.push({ icon: "tag", label: config.alias });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }
  if (config?.kmsKeyArn) {
    items.push({ icon: "key", label: "Custom KMS key" });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
