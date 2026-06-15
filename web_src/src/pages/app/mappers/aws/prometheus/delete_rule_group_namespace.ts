import type { ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../../types";
import type React from "react";
import { stringOrDash } from "../../utils";
import {
  buildPrometheusComponentProps,
  firstOutputData,
  formatExecutionTimestamp,
  prometheusSubtitle,
  ruleGroupNamespaceFromMetadata,
  type RuleGroupNamespaceOutput,
} from "./common";
import { ruleGroupNamespaceMetadataList } from "./get_rule_group_namespace";

export const deleteRuleGroupNamespaceMapper: ComponentBaseMapper = {
  props(context) {
    return buildPrometheusComponentProps(context, ruleGroupNamespaceMetadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const data = firstOutputData<RuleGroupNamespaceOutput>(context.execution.outputs);
    return {
      "Deleted At": stringOrDash(formatExecutionTimestamp(context.execution)),
      Namespace: stringOrDash(data?.namespace ?? ruleGroupNamespaceFromMetadata(context.node)),
      Status: data?.deleted ? "Deleted" : "-",
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return prometheusSubtitle(context);
  },
};
