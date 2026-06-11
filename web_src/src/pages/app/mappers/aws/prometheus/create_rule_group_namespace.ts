import type { ComponentBaseMapper, ExecutionDetailsContext, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  MAX_METADATA_ITEMS,
  prometheusSubtitle,
  type RuleGroupNamespaceOutput,
  ruleGroupNamespaceExecutionDetails,
  workspaceAliasFromMetadata,
} from "./common";

interface CreateRuleGroupNamespaceConfiguration {
  region?: string;
  workspace?: string;
  name?: string;
}

export const createRuleGroupNamespaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, createRuleGroupNamespaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<RuleGroupNamespaceOutput>(context.execution.outputs);
    const config = context.node.configuration as CreateRuleGroupNamespaceConfiguration | undefined;
    return ruleGroupNamespaceExecutionDetails(data?.ruleGroupNamespace, context.execution, {
      timestampLabel: "Created At",
      fallbackName: config?.name,
      timestampSource: "created",
      showStatus: false,
    });
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};

function createRuleGroupNamespaceMetadataList(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as CreateRuleGroupNamespaceConfiguration | undefined;
  const items: MetadataItem[] = [];
  const workspaceAlias = workspaceAliasFromMetadata(node);

  if (config?.name) {
    items.push({ icon: "file-text", label: config.name });
  }
  if (workspaceAlias || config?.workspace) {
    items.push({ icon: "activity", label: workspaceAlias ?? config?.workspace ?? "" });
  }
  if (config?.region) {
    items.push({ icon: "globe", label: config.region });
  }

  return items.slice(0, MAX_METADATA_ITEMS);
}
