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
import type { CommitStatus, GitLabNodeMetadata } from "./types";
import { buildGitlabExecutionSubtitle, shortSha } from "./utils";

interface GetCommitStatusConfiguration {
  project?: string;
  sha?: string;
}

export const getCommitStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as GetCommitStatusConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.sha) {
      metadataItems.push({ icon: "git-commit", label: shortSha(configuration.sha) });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const statuses = outputs?.default?.[0]?.data as CommitStatus[] | undefined;
    return buildGitlabExecutionSubtitle(context.execution, statuses?.[0]?.status);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data || !Array.isArray(payload.data)) {
      return {};
    }

    const statuses = payload.data as CommitStatus[];
    const latest = statuses[0];
    const details: Record<string, string> = {
      "Retrieved At": formatTimestamp(latest?.created_at, payload.timestamp),
      Commit: latest?.sha ? latest.sha.slice(0, 8) : "-",
      Statuses: statuses.length.toString(),
    };

    if (latest?.status) {
      details["Latest State"] = latest.status;
    }

    addDetailIfPresent(details, "Ref", latest?.ref);
    addDetailIfPresent(details, "Status URL", latest?.target_url);

    return details;
  },
};

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
