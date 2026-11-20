import type { ComponentBaseMapper, EventStateRegistry, ExecutionDetailsContext, OutputPayload } from "../types";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase/eventState";
import { buildExecutionSubtitle } from "../eventDisplay";
import { defaultStateFunction } from "../stateRegistry";
import { baseProps } from "./base";

type Reviewer = { uuid: string; displayName?: string };
type ReviewOutput = {
  pullNumber?: number;
  status?: "requested" | "already_covered" | "stale_pr";
  reason?: string;
  headSha?: string;
  candidates?: Reviewer[];
  requested?: Reviewer[];
  alreadyCovered?: Reviewer[];
  unresolvedApprovals?: Reviewer[];
};

function reviewOutput(context: ExecutionDetailsContext): ReviewOutput | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as ReviewOutput | undefined;
}

function reviewerNames(reviewers: Reviewer[]): string {
  return reviewers.map((reviewer) => reviewer.displayName || reviewer.uuid).join(", ");
}

export const reviewCandidatesMapper: ComponentBaseMapper = {
  props: (context) => baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions),
  subtitle: (context) => buildExecutionSubtitle(context.execution),
  getExecutionDetails: (context) => {
    const output = reviewOutput(context);
    const details: Record<string, string> = {
      "Started At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };
    if (output?.pullNumber) details["Pull Request"] = `#${output.pullNumber}`;
    if (output?.headSha) details.Commit = output.headSha;
    if (output?.candidates) details.Candidates = reviewerNames(output.candidates) || "No eligible reviewers";
    return details;
  },
};

export const ensureReviewersMapper: ComponentBaseMapper = {
  ...reviewCandidatesMapper,
  getExecutionDetails: (context) => {
    const details = reviewCandidatesMapper.getExecutionDetails(context) as Record<string, string>;
    const output = reviewOutput(context);
    if (output?.requested?.length) details.Requested = reviewerNames(output.requested);
    if (output?.alreadyCovered?.length) details["Already Covered"] = reviewerNames(output.alreadyCovered);
    if (output?.unresolvedApprovals?.length) {
      details["Unverified Approvals"] = `${reviewerNames(output.unresolvedApprovals)}. The reviewed commit is unknown.`;
    }
    if (output?.reason)
      details.Reason = output.reason === "closed" ? "Pull request is closed" : "Assessed commit changed";
    return details;
  },
};

export const ENSURE_REVIEWERS_STATE_REGISTRY: EventStateRegistry = {
  stateMap: {
    ...DEFAULT_EVENT_STATE_MAP,
    requested: { ...DEFAULT_EVENT_STATE_MAP.success, label: "Requested" },
    already_covered: { ...DEFAULT_EVENT_STATE_MAP.success, label: "Already Covered" },
    stale_pr: { ...DEFAULT_EVENT_STATE_MAP.neutral, label: "Pull Request Changed" },
  },
  getState: (execution) => {
    const state = defaultStateFunction(execution);
    if (state !== "success") return state;
    const outputs = execution?.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as ReviewOutput | undefined;
    return output?.status || state;
  },
};
