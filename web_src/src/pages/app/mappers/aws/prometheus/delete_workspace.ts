import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildPrometheusComponentProps, firstOutputData, MAX_METADATA_ITEMS, prometheusSubtitle } from "./common";

interface DeleteWorkspaceConfiguration {
  region?: string;
  workspace?: string;
}

interface DeleteWorkspaceOutput {
  workspaceId?: string;
  deleted?: boolean;
}

export const deleteWorkspaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, deleteWorkspaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<DeleteWorkspaceOutput>(context.execution.outputs);
    if (!data) {
      return {};
    }

    return {
      "Workspace ID": stringOrDash(data.workspaceId),
      Status: data.deleted ? "Deleted" : "-",
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

function deleteWorkspaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as DeleteWorkspaceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.workspace) {
    items.push({ icon: "activity", label: config.workspace });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
