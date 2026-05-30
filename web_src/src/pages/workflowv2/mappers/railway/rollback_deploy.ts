import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseProps } from "./base";

interface RollbackDeployConfiguration {
  deployId?: string;
}

interface RollbackDeployOutput {
  deployId?: string;
  rolledBack?: boolean;
}

function stringOrDash(value?: unknown): string {
  return value === undefined || value === null || value === "" ? "-" : String(value);
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as RollbackDeployConfiguration | undefined;
  return configuration?.deployId ? [{ icon: "rotate-ccw", label: `Rollback to: ${configuration.deployId}` }] : [];
}

export const rollbackDeployMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions),
      metadata: metadataList(context.node),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return context.execution.createdAt ? renderTimeAgo(new Date(context.execution.createdAt)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as RollbackDeployOutput | undefined;

    return {
      "Deploy ID": stringOrDash(result?.deployId),
      "Rollback Requested": stringOrDash(result?.rolledBack),
    };
  },
};
