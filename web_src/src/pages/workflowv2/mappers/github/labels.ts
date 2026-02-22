import {
  OutputPayload,
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
} from "../types";
import { ComponentBaseProps } from "@/ui/componentBase";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import { Label } from "./types";

export const labelsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs || !outputs.default || outputs.default.length === 0) {
      return details;
    }

    const labels = outputs.default[0].data as Label[];
    if (Array.isArray(labels)) {
      details["Labels"] = labels.map((label) => label.name).join(", ") || "-";
    }

    return details;
  },
};
