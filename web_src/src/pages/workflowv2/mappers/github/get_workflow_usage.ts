import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";

interface WorkflowUsageOutput {
  total_minutes_used: number;
  total_paid_minutes_used: number;
  included_minutes: number;
  minutes_used_breakdown: Record<string, number>;
  organization: string;
}

export const getWorkflowUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (!outputs?.default || outputs.default.length === 0) {
      return "Retrieving usage...";
    }

    const usage = outputs.default[0].data as WorkflowUsageOutput;
    if (!usage) {
      return "Usage retrieved";
    }

    return `${usage.total_minutes_used.toFixed(1)} minutes used`;
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const usage = outputs.default[0].data as WorkflowUsageOutput;
    if (!usage) {
      return details;
    }

    details["Organization"] = usage.organization || "-";
    details["Total Minutes Used"] = usage.total_minutes_used?.toFixed(2) || "-";
    details["Paid Minutes Used"] = usage.total_paid_minutes_used?.toFixed(2) || "-";
    details["Included Minutes"] = usage.included_minutes?.toString() || "-";

    // Add breakdown by OS
    if (usage.minutes_used_breakdown) {
      const breakdown = Object.entries(usage.minutes_used_breakdown)
        .map(([os, minutes]) => `${os}: ${minutes}`)
        .join(", ");
      details["Usage by OS"] = breakdown || "-";
    }

    return details;
  },
};
