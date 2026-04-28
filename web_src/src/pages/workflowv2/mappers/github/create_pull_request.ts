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

export const createPullRequestMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string | ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const pr = outputs.default[0].data as PullRequest;
    details["Created At"] = context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-";
    if (pr?.number !== undefined) {
      details["Pull Request"] = `#${pr.number}`;
    }
    if (pr?.title) {
      details["Title"] = pr.title;
    }
    if (pr?.state) {
      details["State"] = pr.draft ? `${pr.state} (draft)` : pr.state;
    }
    if (pr?.head?.ref && pr?.base?.ref) {
      details["Branches"] = `${pr.head.ref} → ${pr.base.ref}`;
    }
    details["Pull Request URL"] = pr?.html_url || "";

    return details;
  },
};
