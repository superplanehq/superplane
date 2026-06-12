import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  MAX_METADATA_ITEMS,
  prometheusSubtitle,
  ruleGroupNamespaceExecutionDetails,
  ruleGroupNamespaceFromMetadata,
  type RuleGroupNamespaceOutput,
  workspaceAliasFromMetadata,
} from "./common";

interface RuleGroupNamespaceConfiguration {
  region?: string;
  workspace?: string;
  namespace?: string;
}

export const getRuleGroupNamespaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, ruleGroupNamespaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<RuleGroupNamespaceOutput>(context.execution.outputs);
    return ruleGroupNamespaceExecutionDetails(data?.ruleGroupNamespace, context.execution, {
      timestampLabel: "Retrieved At",
      fallbackName: ruleGroupNamespaceFromMetadata(context.node),
    });
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

export function ruleGroupNamespaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as RuleGroupNamespaceConfiguration | undefined;
  const items: MetadataItem[] = [];
  const namespace = ruleGroupNamespaceFromMetadata(node) ?? config?.namespace;
  const workspaceAlias = workspaceAliasFromMetadata(node);

  if (namespace) {
    items.push({ icon: "file-text", label: namespace });
  }
  if (workspaceAlias || config?.workspace) {
    items.push({ icon: "activity", label: workspaceAlias ?? config?.workspace ?? "" });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
