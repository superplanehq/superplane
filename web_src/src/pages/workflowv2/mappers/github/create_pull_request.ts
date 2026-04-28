import type { ReactNode } from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGithubExecutionSubtitle } from "./utils";
import type { PullRequest } from "./types";

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

export const createPullRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const pr = outputs?.default?.[0]?.data as PullRequest | undefined;

    const details: Record<string, string> = {
      "Created At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
    };

    for (const [key, value] of pullRequestDetailFields(pr)) {
      details[key] = value;
    }

    return details;
  },
};
