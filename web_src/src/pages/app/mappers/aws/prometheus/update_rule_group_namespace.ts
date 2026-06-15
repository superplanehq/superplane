import type { ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../../types";
import type React from "react";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  prometheusSubtitle,
  type RuleGroupNamespaceOutput,
  ruleGroupNamespaceExecutionDetails,
  ruleGroupNamespaceFromMetadata,
} from "./common";
import { ruleGroupNamespaceMetadataList } from "./get_rule_group_namespace";

export const updateRuleGroupNamespaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, ruleGroupNamespaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<RuleGroupNamespaceOutput>(context.execution.outputs);
    return ruleGroupNamespaceExecutionDetails(data?.ruleGroupNamespace, context.execution, {
      timestampLabel: "Updated At",
      fallbackName: ruleGroupNamespaceFromMetadata(context.node),
      timestampSource: "completed",
      showStatus: false,
    });
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};
