import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";

import { baseProps } from "./base";

export interface ProjectRefConfiguration {
  project?: string;
}

/** Shared: subtitle renderer used by all Vercel component mappers. */
export function executionSubtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

/** Shared: base props plus configuration metadata for a Vercel component. */
export function propsWithMetadata(
  context: ComponentBaseContext,
  metadataList: (node: NodeInfo) => MetadataItem[],
): ComponentBaseProps {
  const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  return { ...base, metadata: metadataList(context.node) };
}

/** Shared: read the default output payload of an execution. */
export function readDefaultOutput(context: ExecutionDetailsContext): OutputPayload | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0];
}

export function projectMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as ProjectRefConfiguration | undefined;
  if (!configuration?.project) {
    return [];
  }
  return [{ icon: "database", label: `Project: ${configuration.project}` }];
}
