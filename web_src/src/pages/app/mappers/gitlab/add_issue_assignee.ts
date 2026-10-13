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
import { baseProps } from "./base";
import type { GitLabNodeMetadata, Issue } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";
import { buildAssigneeDetails } from "./assignee_utils";

interface AddIssueAssigneeConfiguration {
  project?: string;
  issueIid?: string;
}

export const addIssueAssigneeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as AddIssueAssigneeConfiguration | undefined) ?? {};
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
    return buildGitlabExecutionSubtitle(context.execution, "Assignee Added");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (!payload?.data) {
      return {};
    }

    return buildAssigneeDetails(payload.data as Issue, payload.timestamp);
  },
};
