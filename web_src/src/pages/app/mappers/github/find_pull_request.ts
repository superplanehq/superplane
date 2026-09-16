import type { ReactNode } from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import type { PullRequest } from "./types";

type FindPullRequestOutputs = { found?: OutputPayload[]; notFound?: OutputPayload[] };

const FIND_PULL_REQUEST_STATE_MAP = {
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
};

const findPullRequestStateFunction: StateFunction = (execution) => {
  if (!execution) return "neutral";

  const outputs = execution.outputs as FindPullRequestOutputs | undefined;

  if (outputs?.found?.length) {
    return "found";
  }

  if (outputs?.notFound?.length) {
    return "notFound";
  }

  return "neutral";
};

export const FIND_PULL_REQUEST_STATE_REGISTRY: EventStateRegistry = {
  stateMap: FIND_PULL_REQUEST_STATE_MAP,
  getState: findPullRequestStateFunction,
};

function pullRequestDetailFields(pr: PullRequest | undefined): Array<[string, string]> {
  if (!pr) return [];

  const branches = pr.head?.ref && pr.base?.ref ? `${pr.head.ref} → ${pr.base.ref}` : "";
  const state = pr.state ? (pr.draft ? `${pr.state} (draft)` : pr.state) : "";

  return (
    [
      ["Pull Request", pr.number !== undefined ? `#${pr.number}` : ""],
      ["Title", pr.title ?? ""],
      ["State", state],
      ["Branches", branches],
      ["Pull Request URL", pr.html_url ?? ""],
    ] as Array<[string, string]>
  ).filter(([, value]) => value !== "");
}

export const findPullRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as FindPullRequestOutputs | undefined;
    const pr = outputs?.found?.[0]?.data as PullRequest | undefined;

    const details: Record<string, string> = {
      "Created At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      Result: outputs?.found?.length ? "Found" : "Not Found",
    };

    for (const [key, value] of pullRequestDetailFields(pr)) {
      details[key] = value;
    }

    return details;
  },
};
