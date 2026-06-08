import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import { buildPrometheusComponentProps, firstOutputData, MAX_METADATA_ITEMS, prometheusSubtitle } from "./common";

interface UpdateWorkspaceConfiguration {
  region?: string;
  workspace?: string;
  alias?: string;
}

interface UpdateWorkspaceOutput {
  workspaceId?: string;
  alias?: string;
  updated?: boolean;
}

export const updateWorkspaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, updateWorkspaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<UpdateWorkspaceOutput>(context.execution.outputs);
    if (!data) {
      return {};
    }

    return {
      "Workspace ID": stringOrDash(data.workspaceId),
      Alias: stringOrDash(data.alias),
      Status: data.updated ? "Updated" : "-",
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

function updateWorkspaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as UpdateWorkspaceConfiguration | undefined;
  const items: MetadataItem[] = [];

  if (config?.workspace) {
    items.push({ icon: "activity", label: config.workspace });
  }
  if (config?.alias) {
    items.push({ icon: "tag", label: config.alias });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
