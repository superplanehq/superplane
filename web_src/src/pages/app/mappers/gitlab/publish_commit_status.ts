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

interface PublishCommitStatusConfiguration {
  project?: string;
  sha?: string;
  state?: string;
}

export const publishCommitStatusMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as PublishCommitStatusConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.sha) {
      metadataItems.push({ icon: "git-commit", label: shortSha(configuration.sha) });
    }

    if (configuration.state) {
      metadataItems.push({ icon: "activity", label: configuration.state });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const status = outputs?.default?.[0]?.data as CommitStatus | undefined;
    return buildGitlabExecutionSubtitle(context.execution, status?.status);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data) {
      return {};
    }

    const status = payload.data as CommitStatus;
    const details: Record<string, string> = {
      "Published At": formatTimestamp(status.created_at, payload.timestamp),
      Commit: status.sha ? status.sha.slice(0, 8) : "-",
      State: status.status || "-",
    };

    addDetailIfPresent(details, "Context", status.name);
    addDetailIfPresent(details, "Status URL", status.target_url);
    addDetailIfPresent(details, "Description", status.description);

    return details;
  },
};

function addDetailIfPresent(details: Record<string, string>, label: string, value?: string) {
  if (value) {
    details[label] = value;
  }
}
