import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  MAX_METADATA_ITEMS,
  prometheusSubtitle,
  ruleGroupNamespaceExecutionDetails,
  type RuleGroupNamespaceOutput,
  workspaceAliasFromMetadata,
} from "./common";

interface UpdateRuleGroupNamespaceConfiguration {
  region?: string;
  workspace?: string;
  namespace?: string;
}

export const updateRuleGroupNamespaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, updateRuleGroupNamespaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<RuleGroupNamespaceOutput>(context.execution.outputs);
    const config = context.node.configuration as UpdateRuleGroupNamespaceConfiguration | undefined;
    return ruleGroupNamespaceExecutionDetails(data?.namespace, context.execution, {
      timestampLabel: "Updated At",
      fallbackName: config?.namespace,
      fallbackWorkspaceAlias: workspaceAliasFromMetadata(context.node) ?? config?.workspace,
    });
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

function updateRuleGroupNamespaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as UpdateRuleGroupNamespaceConfiguration | undefined;
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
