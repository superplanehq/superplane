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

interface DeleteRuleGroupNamespaceConfiguration {
  region?: string;
  workspace?: string;
  namespace?: string;
}

interface DeleteRuleGroupNamespaceOutput {
  workspaceId?: string;
  namespace?: string;
  deleted?: boolean;
}

export const deleteRuleGroupNamespaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, deleteRuleGroupNamespaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<DeleteRuleGroupNamespaceOutput>(context.execution.outputs);
    const config = context.node.configuration as DeleteRuleGroupNamespaceConfiguration | undefined;
    const workspace = workspaceAliasFromMetadata(context.node) ?? config?.workspace;

    if (!data) {
      return {
        "Deleted At": stringOrDash(formatExecutionTimestamp(context.execution)),
        Name: stringOrDash(config?.namespace),
        Workspace: stringOrDash(workspace),
        Status: "-",
      };
    }

    return {
      "Deleted At": stringOrDash(formatExecutionTimestamp(context.execution)),
      Name: stringOrDash(data.namespace ?? config?.namespace),
      Workspace: stringOrDash(workspace),
      Status: data.deleted ? "Deleted" : "-",
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

function deleteRuleGroupNamespaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as DeleteRuleGroupNamespaceConfiguration | undefined;
  const items: MetadataItem[] = [];
  const workspaceAlias = workspaceAliasFromMetadata(node);

  if (config?.namespace) {
    items.push({ icon: "tag", label: config.namespace });
  }
  if (workspaceAlias || config?.workspace) {
    items.push({ icon: "activity", label: workspaceAlias ?? config?.workspace ?? "" });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
