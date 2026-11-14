import type { ComponentBaseMapper, EventStateRegistry, OutputPayload } from "../types";
import { buildActionStateRegistry, buildExecutionSubtitle } from "../eventDisplay";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase/eventState";
import { baseProps } from "./base";

interface ReviewCheckOutput {
  status?: string;
  state?: string;
  sha?: string;
  reason?: string;
  reviewers?: { uuid: string; reason: string }[];
}

const appliedRegistry = buildActionStateRegistry("applied");
export const REVIEW_CHECK_STATE_REGISTRY: EventStateRegistry = {
  stateMap: {
    ...appliedRegistry.stateMap,
    registered: { ...DEFAULT_EVENT_STATE_MAP.neutral, label: "Registered" },
    skipped: { ...DEFAULT_EVENT_STATE_MAP.neutral, label: "Skipped" },
    superseded: { ...DEFAULT_EVENT_STATE_MAP.neutral, label: "Superseded" },
  },
  getState(execution) {
    const state = appliedRegistry.getState(execution);
    if (state !== "applied") return state;
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as ReviewCheckOutput | undefined;
    if (output?.status === "registered" || output?.status === "skipped" || output?.status === "superseded")
      return output.status;
    return state;
  },
};

export const setPullRequestReviewCheckMapper: ComponentBaseMapper = {
  props(context) {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },
  subtitle(context) {
    return buildExecutionSubtitle(context.execution);
  },
  getExecutionDetails(context) {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as ReviewCheckOutput | undefined;
    const config = (context.execution.configuration || context.node.configuration || {}) as {
      repository?: string;
      pullRequestId?: string;
      resultUrl?: string;
    };
    return {
      "Started at": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      Repository: config.repository || "-",
      "Pull request": config.pullRequestId || "-",
      ...reviewCheckDetails(output),
      ...(config.resultUrl ? { "Run URL": config.resultUrl } : {}),
    };
  },
};

function reviewCheckDetails(output: ReviewCheckOutput | undefined): Record<string, string> {
  return {
    Result: output?.status || "-",
    "Commit status": output?.state || "-",
    Commit: output?.sha || "-",
    Reason: output?.reason?.replace(/_/g, " ") || "-",
    ...(output?.reviewers?.length
      ? {
          Reviewers: output.reviewers
            .map((reviewer) => `${reviewer.uuid}: ${reviewer.reason.replace(/_/g, " ")}`)
            .join(", "),
        }
      : {}),
  };
}
