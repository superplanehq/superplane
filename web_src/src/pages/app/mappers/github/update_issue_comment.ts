import type { ComponentBaseProps } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import type { BaseNodeMetadata, Comment } from "./types";
import { buildGithubExecutionSubtitle } from "./utils";
import { integrationResourceDisplayLabel } from "@/lib/integrationResourceLabel";

interface UpdateIssueCommentConfiguration {
  repository?: unknown;
}

export const updateIssueCommentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as UpdateIssueCommentConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as BaseNodeMetadata | undefined) ?? ({} as BaseNodeMetadata);

    const repository = integrationResourceDisplayLabel(configuration.repository) || metadata?.repository?.name;
    const metadataItems: MetadataItem[] = [];

    if (repository) {
      metadataItems.push({ icon: "book", label: repository });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGithubExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const comment = outputs.default[0].data as Comment;
    details["Updated At"] = comment?.updated_at ? new Date(comment.updated_at).toLocaleString() : "-";
    details["URL"] = comment?.html_url || "-";

    return details;
  },
};
