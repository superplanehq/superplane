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
import type { AwardEmoji, GitLabNodeMetadata } from "./types";
import { buildGitlabExecutionSubtitle } from "./utils";

interface AddReactionConfiguration {
  project?: string;
  mergeRequestIid?: string;
  target?: string;
  noteId?: string;
  content?: string;
}

function getReactionDisplay(name?: string): string {
  switch (name) {
    case "thumbsup":
      return "👍";
    case "thumbsdown":
      return "👎";
    case "laughing":
      return "😄";
    case "confused":
      return "😕";
    case "heart":
      return "❤️";
    case "tada":
      return "🎉";
    case "rocket":
      return "🚀";
    case "eyes":
      return "👀";
    default:
      return name || "-";
  }
}

export const addReactionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as AddReactionConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as GitLabNodeMetadata | undefined) ?? ({} as GitLabNodeMetadata);

    const project = metadata?.project?.name || configuration.project;
    const metadataItems: MetadataItem[] = [];

    if (project) {
      metadataItems.push({ icon: "book", label: project });
    }

    if (configuration.mergeRequestIid) {
      metadataItems.push({ icon: "git-pull-request", label: `!${configuration.mergeRequestIid}` });
    }

    if (configuration.target === "note" && configuration.noteId) {
      metadataItems.push({ icon: "message-square", label: `Comment #${configuration.noteId}` });
    }

    if (configuration.content) {
      metadataItems.push({ icon: "smile", label: `Reaction: ${getReactionDisplay(configuration.content)}` });
    }

    return {
      ...props,
      metadata: metadataItems.length > 0 ? metadataItems : props.metadata,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildGitlabExecutionSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default || outputs.default.length === 0) {
      return details;
    }

    const awardEmoji = outputs.default[0].data as AwardEmoji;
    details["Reaction"] = getReactionDisplay(awardEmoji?.name);
    details["Created By"] = awardEmoji?.user?.username || "-";
    details["Created At"] = awardEmoji?.created_at ? new Date(awardEmoji.created_at).toLocaleString() : "-";

    return details;
  },
};
