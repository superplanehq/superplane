import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../../utils";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  formatExecutionTimestamp,
  MAX_METADATA_ITEMS,
  prometheusSubtitle,
  workspaceAliasFromMetadata,
} from "./common";

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
    const config = context.node.configuration as UpdateWorkspaceConfiguration | undefined;
    if (!data) {
      return {
        "Updated At": stringOrDash(formatExecutionTimestamp(context.execution)),
        Alias: stringOrDash(config?.alias ?? workspaceAliasFromMetadata(context.node)),
        Status: "-",
      };
    }

    return {
      "Updated At": stringOrDash(formatExecutionTimestamp(context.execution)),
      Alias: stringOrDash(data.alias ?? config?.alias ?? workspaceAliasFromMetadata(context.node)),
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
  const currentAlias = workspaceAliasFromMetadata(node);
  const newAlias = config?.alias?.trim();

  if (currentAlias || config?.workspace) {
    items.push({ icon: "activity", label: currentAlias ? `Current: ${currentAlias}` : (config?.workspace ?? "") });
  }
  if (newAlias) {
    items.push({ icon: "tag", label: `New: ${newAlias}` });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
