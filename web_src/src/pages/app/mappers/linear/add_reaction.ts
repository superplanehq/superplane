import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { linearComponentBaseProps } from "./base";
import { buildReactionDetails, buildReactionSubtitle } from "./utils";
import type { AddReactionConfiguration } from "./types";

export const addReactionMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return linearComponentBaseProps(context, metadataList(context.node));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    return buildReactionDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildReactionSubtitle(context.execution);
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as AddReactionConfiguration | undefined;

  const target = configuration?.target === "comment" ? configuration?.comment : configuration?.issue;
  if (target) {
    metadata.push({ icon: "hash", label: target });
  }

  if (configuration?.emoji) {
    metadata.push({ icon: "smile", label: configuration.emoji });
  }

  return metadata;
}
