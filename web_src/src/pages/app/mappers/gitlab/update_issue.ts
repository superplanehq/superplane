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
import { getSummaryDetailsForIssue } from "./issue_utils";

interface UpdateIssueConfiguration {
  project?: string;
  issueIid?: string;
  state?: string;
}

export const updateIssueMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as UpdateIssueConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.issueIid) {
      metadataItems.push({ icon: "circle-dot", label: `#${configuration.issueIid}` });
    }

    if (configuration.state) {
      metadataItems.push({ icon: "refresh-cw", label: configuration.state === "close" ? "Close" : "Reopen" });
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
    return buildGitlabExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

    if (!outputs?.default || outputs.default.length === 0) {
      return {};
    }

    const issue = outputs.default[0].data as Issue;
    return getSummaryDetailsForIssue(context.execution, issue);
  },
};
