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
import { Release } from "./types";

export const listReleasesMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    const list = outputs?.default || [];
    details["Retrieved At"] = context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-";
    details["Releases"] = list.length.toString();

    if (list.length > 0) {
      const first = list[0].data as Release;
      details["First Tag"] = first?.tag_name || "";
      details["First Release URL"] = first?.html_url || "";
      if (first?.name) {
        details["First Release Name"] = first.name;
      }
    }

    return details;
  },
};

