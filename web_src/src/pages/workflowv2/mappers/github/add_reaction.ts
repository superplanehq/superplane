import type React from "react";
import type { ComponentBaseProps } from "@/ui/componentBase";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import type { BaseNodeMetadata } from "./types";
import { buildGithubExecutionSubtitle } from "./utils";

interface AddReactionConfiguration {
  repository?: string;
  content?: string;
}

interface AddReactionOutput {
  created_at?: string;
  content?: string;
  user?: {
    login?: string;
  };
}

function getReactionDisplay(content?: string): string {
  switch (content) {
    case "+1":
      return "👍";
    case "-1":
      return "👎";
    case "laugh":
      return "😄";
    case "confused":
      return "😕";
    case "heart":
      return "❤️";
    case "hooray":
      return "🎉";
    case "rocket":
      return "🚀";
    case "eyes":
      return "👀";
    default:
      return content || "-";
  }
}

export const addReactionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const props = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    const configuration = (context.node.configuration as AddReactionConfiguration | undefined) ?? {};
    const metadata = (context.node.metadata as BaseNodeMetadata | undefined) ?? ({} as BaseNodeMetadata);

    const repository = configuration.repository || metadata?.repository?.name;
    const reaction = configuration.content;
    const metadataItems = [];

    if (repository) {
      metadataItems.push({
        icon: "book",
        label: repository,
      });
    }

    if (reaction) {
      metadataItems.push({
        icon: "smile",
        label: `Reaction: ${getReactionDisplay(reaction)}`,
      });
    }

    return {
      ...props,
      metadata: metadataItems,
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

    const reaction = outputs.default[0].data as AddReactionOutput;
    details["Reaction"] = getReactionDisplay(reaction?.content);
    details["Created By"] = reaction?.user?.login || "-";
    details["Created At"] = reaction?.created_at ? new Date(reaction.created_at).toLocaleString() : "-";

    return details;
  },
};
