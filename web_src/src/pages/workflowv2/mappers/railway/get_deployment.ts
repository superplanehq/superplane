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

interface GetDeploymentConfiguration {
  deployId?: string;
}

interface GetDeploymentOutput {
  deployId?: string;
  status?: string;
  projectId?: string;
  serviceId?: string;
  environmentId?: string;
  createdAt?: string;
  updatedAt?: string;
  canRollback?: boolean;
  canRedeploy?: boolean;
}

function stringOrDash(value?: unknown): string {
  return value === undefined || value === null || value === "" ? "-" : String(value);
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as GetDeploymentConfiguration | undefined;
  return configuration?.deployId ? [{ icon: "hash", label: `Deploy: ${configuration.deployId}` }] : [];
}

export const getDeploymentMapper: ComponentBaseMapper = {
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
    const result = outputs?.default?.[0]?.data as GetDeploymentOutput | undefined;

    return {
      "Deploy ID": stringOrDash(result?.deployId),
      Status: stringOrDash(result?.status),
      "Project ID": stringOrDash(result?.projectId),
      "Service ID": stringOrDash(result?.serviceId),
      "Environment ID": stringOrDash(result?.environmentId),
      "Created At": stringOrDash(result?.createdAt),
      "Updated At": stringOrDash(result?.updatedAt),
      "Can Rollback": stringOrDash(result?.canRollback),
      "Can Redeploy": stringOrDash(result?.canRedeploy),
    };
  },
};
