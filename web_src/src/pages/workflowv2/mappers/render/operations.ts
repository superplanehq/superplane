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
import { stringOrDash } from "./common";
import { baseProps } from "./base";

type RenderConfiguration = {
  service?: string;
  resources?: string[];
  metricTypes?: string[];
  statuses?: string[];
  autoDeploy?: string;
  minInstances?: number;
  maxInstances?: number;
  startCommand?: string;
  jobId?: string;
  limit?: number;
};

type RenderOutput = Record<string, unknown>;

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as RenderConfiguration | undefined;

  if (configuration?.service) {
    metadata.push({ icon: "server", label: `Service: ${configuration.service}` });
  }
  if (configuration?.resources?.length) {
    metadata.push({ icon: "server", label: `Resources: ${configuration.resources.length}` });
  }
  if (configuration?.metricTypes?.length) {
    metadata.push({ icon: "activity", label: `Metrics: ${configuration.metricTypes.join(", ")}` });
  }
  if (configuration?.statuses?.length) {
    metadata.push({ icon: "filter", label: `Statuses: ${configuration.statuses.join(", ")}` });
  }
  if (configuration?.autoDeploy) {
    metadata.push({ icon: "git-branch", label: `Auto deploy: ${configuration.autoDeploy}` });
  }
  if (configuration?.minInstances || configuration?.maxInstances) {
    metadata.push({
      icon: "trending-up",
      label: `Autoscale: ${configuration.minInstances ?? "?"}-${configuration.maxInstances ?? "?"}`,
    });
  }
  if (configuration?.startCommand) {
    metadata.push({ icon: "terminal", label: "One-off job" });
  }
  if (configuration?.jobId) {
    metadata.push({ icon: "terminal", label: `Job: ${configuration.jobId}` });
  }
  if (configuration?.limit) {
    metadata.push({ icon: "list", label: `Limit: ${configuration.limit}` });
  }

  return metadata;
}

function firstDefaultOutput(context: ExecutionDetailsContext): RenderOutput | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as RenderOutput | undefined;
}

export const renderOperationMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const base = baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
    return { ...base, metadata: metadataList(context.node) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const result = firstDefaultOutput(context);
    return {
      "Executed At": context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-",
      "Service ID": stringOrDash(result?.serviceId),
      "Job ID": stringOrDash(result?.jobId),
      Status: stringOrDash(result?.status),
      Count: stringOrDash(result?.count),
      "Error Count": stringOrDash(result?.errorCount),
      "Auto Deploy": stringOrDash(result?.autoDeploy),
      Resources: Array.isArray(result?.resources) ? result.resources.join(", ") : stringOrDash(result?.resources),
    };
  },
};
