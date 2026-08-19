import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { formatTimestamp } from "../utils";
import { baseProps } from "./base";
import type { MergeRequest } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

export const createMergeRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default?.[0]?.data) {
      const mergeRequest = outputs.default[0].data as MergeRequest;
      return `!${mergeRequest.iid} ${mergeRequest.title}`.trim();
    }
    return buildGitlabExecutionSubtitle(context.execution, "Merge Request Created");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data) {
      return {};
    }

    const mergeRequest = payload.data as MergeRequest;
    const details: Record<string, string> = {
      "Created At": formatTimestamp(mergeRequest.created_at, payload.timestamp),
      "Merge Request": mergeRequest.iid ? `!${mergeRequest.iid} ${mergeRequest.title || ""}`.trim() : "-",
    };

    addDetailIfPresent(details, "Merge Request URL", mergeRequest.web_url);
    addDetailIfPresent(details, "Source Branch", mergeRequest.source_branch);
    addDetailIfPresent(details, "Target Branch", mergeRequest.target_branch);
    addDetailIfPresent(details, "State", mergeRequest.state);

    return details;
  },
};

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
