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

interface DeleteWorkspaceConfiguration {
  region?: string;
  workspace?: string;
}

interface DeleteWorkspaceOutput {
  alias?: string;
  workspaceId?: string;
  deleted?: boolean;
}

export const deleteWorkspaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, deleteWorkspaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<DeleteWorkspaceOutput>(context.execution.outputs);
    const config = context.node.configuration as DeleteWorkspaceConfiguration | undefined;
    if (!data) {
      return {
        "Deleted At": stringOrDash(formatExecutionTimestamp(context.execution)),
        Alias: stringOrDash(workspaceAliasFromMetadata(context.node) ?? config?.workspace),
        Status: "-",
      };
    }

    return {
      "Deleted At": stringOrDash(formatExecutionTimestamp(context.execution)),
      Alias: stringOrDash(data.alias ?? workspaceAliasFromMetadata(context.node) ?? config?.workspace),
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
  const alias = workspaceAliasFromMetadata(node);

  if (alias || config?.workspace) {
    items.push({ icon: "activity", label: alias ?? config?.workspace ?? "" });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
