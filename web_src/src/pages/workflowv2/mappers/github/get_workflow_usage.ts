import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { stringOrDash } from "../utils";

type WorkflowUsageOutput = {
  minutes_used?: number;
  minutes_used_breakdown?: Record<string, number>;
  net_amount?: number;
  product?: string;
  year?: number;
  month?: number;
  day?: number;
};

export const getWorkflowUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default?.length) {
      return details;
    }

    const out = outputs.default[0]?.data as WorkflowUsageOutput | undefined;
    if (!out) {
      return details;
    }

    details["Retrieved At"] = context.execution.createdAt
      ? new Date(context.execution.createdAt).toLocaleString()
      : "-";
    details["Product"] = stringOrDash(out.product);

    if (out.year) details["Year"] = String(out.year);
    if (out.month) details["Month"] = String(out.month);
    if (out.day) details["Day"] = String(out.day);

    if (out.minutes_used !== undefined) {
      details["Minutes Used"] = String(out.minutes_used);
    }

    if (out.net_amount !== undefined) {
      details["Net Amount"] = String(out.net_amount);
    }

    if (out.minutes_used_breakdown && Object.keys(out.minutes_used_breakdown).length > 0) {
      // Flatten breakdown into a single line for compactness.
      details["Minutes Breakdown"] = Object.entries(out.minutes_used_breakdown)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([k, v]) => `${k}: ${v}`)
        .join(", ");
    }

    return details;
  },
};
