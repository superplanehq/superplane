import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  MAX_METADATA_ITEMS,
  prometheusSubtitle,
  type WorkspaceOutput,
  workspaceAliasFromMetadata,
  workspaceExecutionDetails,
} from "./common";

interface GetWorkspaceConfiguration {
  region?: string;
  workspace?: string;
}

export const getWorkspaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, getWorkspaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<WorkspaceOutput>(context.execution.outputs);
    return workspaceExecutionDetails(
      data?.workspace,
      context.execution,
      "Retrieved At",
      workspaceAliasFromMetadata(context.node),
    );
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

function getWorkspaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as GetWorkspaceConfiguration | undefined;
  const items: MetadataItem[] = [];
  const alias = workspaceAliasFromMetadata(node);

  if (alias || config?.workspace) {
    items.push({ icon: "activity", label: alias ?? config?.workspace ?? "" });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
