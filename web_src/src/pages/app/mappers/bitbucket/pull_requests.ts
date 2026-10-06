import type { ReactNode } from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase/eventState";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import { buildExecutionSubtitle } from "../eventDisplay";
import { baseProps } from "./base";
import type { BitbucketPullRequest, BitbucketPullRequestComment } from "./types";

type PullRequestOutputs = { default?: OutputPayload[]; found?: OutputPayload[]; notFound?: OutputPayload[] };

export function pullRequestDetailFields(pr: BitbucketPullRequest | undefined): Record<string, string> {
  if (!pr) return {};

  const source = pr.source?.branch?.name;
  const destination = pr.destination?.branch?.name;
  const state = pr.state ? (pr.draft ? `${pr.state} (draft)` : pr.state) : "";
  const fields: Array<[string, string]> = [
    ["Pull Request", pr.id !== undefined ? `#${pr.id}` : ""],
    ["Title", pr.title ?? ""],
    ["State", state],
    ["Branches", source && destination ? `${source} → ${destination}` : ""],
    ["Pull Request URL", pr.links?.html?.href ?? ""],
  ];
  return Object.fromEntries(fields.filter(([, value]) => value !== ""));
}

function createdAtDetail(context: ExecutionDetailsContext): Record<string, string> {
  const createdAt = context.execution.createdAt;
  return { "Created At": createdAt ? new Date(createdAt).toLocaleString() : "-" };
}

function pullRequestMapper(channel: "default" | "found"): ComponentBaseMapper {
  return {
    props(context: ComponentBaseContext): ComponentBaseProps {
      return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    },

    subtitle(context: SubtitleContext): string | ReactNode {
      return buildExecutionSubtitle(context.execution);
    },

    getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
      const outputs = context.execution.outputs as PullRequestOutputs | undefined;
      const pr = outputs?.[channel]?.[0]?.data as BitbucketPullRequest | undefined;
      const details = createdAtDetail(context);
      if (channel === "found") {
        details.Result = outputs?.found?.length ? "Found" : "Not Found";
      }
      return { ...details, ...pullRequestDetailFields(pr) };
    },
  };
}

export const createPullRequestMapper = pullRequestMapper("default");
export const updatePullRequestMapper = pullRequestMapper("default");
export const findPullRequestMapper = pullRequestMapper("found");

export const createPullRequestCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | ReactNode {
    return buildExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as PullRequestOutputs | undefined;
    const comment = outputs?.default?.[0]?.data as BitbucketPullRequestComment | undefined;
    const details = createdAtDetail(context);
    if (comment?.links?.html?.href) {
      details["Comment URL"] = comment.links.html.href;
    }
    return details;
  },
};

const findPullRequestState: StateFunction = (execution) => {
  if (!execution) return "neutral";
  const outputs = execution.outputs as PullRequestOutputs | undefined;
  if (outputs?.found?.length) return "found";
  if (outputs?.notFound?.length) return "notFound";
  return "neutral";
};

export const FIND_PULL_REQUEST_STATE_REGISTRY: EventStateRegistry = {
  stateMap: {
    ...DEFAULT_EVENT_STATE_MAP,
    found: {
      icon: "git-pull-request",
      textColor: "text-gray-800",
      backgroundColor: "bg-green-100",
      badgeColor: "bg-emerald-500",
      label: "Found",
    },
    notFound: {
      icon: "git-pull-request-closed",
      textColor: "text-gray-800",
      backgroundColor: "bg-gray-100",
      badgeColor: "bg-gray-500",
      label: "Not Found",
    },
  },
  getState: findPullRequestState,
};
