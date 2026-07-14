import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { formatTimestamp } from "../utils";
import { baseProps } from "./base";
import type { GitLabNodeMetadata, Issue } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface GetIssueConfiguration {
  project?: string;
  issueIid?: string;
}

export const getIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as GetIssueConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.issueIid) {
      metadataItems.push({ icon: "circle-dot", label: `#${configuration.issueIid}` });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    if (outputs?.default?.[0]?.data) {
      const issue = outputs.default[0].data as Issue;
      return `#${issue.iid} ${issue.title}`;
    }
    return buildGitlabExecutionSubtitle(context.execution, "Issue Retrieved");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload) {
      return {};
    }

    return buildIssueDetails((payload.data ?? {}) as Issue, payload.timestamp);
  },
};

function buildIssueDetails(issue: Issue, payloadTimestamp?: string): Record<string, string> {
  const details: Record<string, string> = {
    "Retrieved At": formatTimestamp(payloadTimestamp),
    Issue: issue.iid ? `#${issue.iid} ${issue.title || ""}`.trim() : "-",
  };

  const labels = (issue.labels ?? []).join(", ");
  addDetailIfPresent(details, "Issue URL", issue.web_url);
  addDetailIfPresent(details, "State", issue.state);
  addDetailIfPresent(details, "Author", issue.author?.username);
  addDetailIfPresent(details, "Labels", labels);

  return details;
}

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
