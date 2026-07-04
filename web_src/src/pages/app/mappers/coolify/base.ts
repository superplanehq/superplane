import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { noopMapper } from "../noop";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

type CoolifyConfiguration = {
  application?: unknown;
  service?: unknown;
  operation?: unknown;
  force?: unknown;
};

function getConfigValue(value: unknown): string | undefined {
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  }
  if (typeof value === "number") {
    return String(value);
  }
  if (!value || typeof value !== "object") {
    return undefined;
  }

  const objectValue = value as Record<string, unknown>;
  for (const field of ["label", "displayName", "name", "value", "id"]) {
    const candidate = objectValue[field];
    if (typeof candidate === "string") {
      const trimmed = candidate.trim();
      if (trimmed.length > 0) {
        return trimmed;
      }
    }
  }
  return undefined;
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = (node.configuration as CoolifyConfiguration | undefined) ?? {};

  const application = getConfigValue(configuration.application);
  const service = getConfigValue(configuration.service);
  const operation = getConfigValue(configuration.operation);

  if (application) {
    metadata.push({ icon: "box", label: `Application: ${application}` });
  }
  if (service) {
    metadata.push({ icon: "server", label: `Service: ${service}` });
  }
  if (operation) {
    metadata.push({ icon: "play", label: `Operation: ${operation}` });
  }
  if (configuration.force === true) {
    metadata.push({ icon: "refresh-cw", label: "Force rebuild" });
  }

  return metadata;
}

function getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  const outputs = context.execution.outputs as { default?: Array<{ data?: Record<string, unknown> }> } | undefined;
  const data = outputs?.default?.[0]?.data;

  if (data) {
    if (typeof data.applicationUuid === "string") {
      details["Application UUID"] = data.applicationUuid;
    }
    if (typeof data.serviceUuid === "string") {
      details["Service UUID"] = data.serviceUuid;
    }
    if (typeof data.operation === "string") {
      details["Operation"] = data.operation;
    }
    if (typeof data.deploymentUuid === "string") {
      details["Deployment UUID"] = data.deploymentUuid;
    }
    if (typeof data.message === "string" && data.message) {
      details["Message"] = data.message;
    }
    if (typeof data.count === "number") {
      details["Count"] = String(data.count);
    }
  }

  if (context.execution.createdAt) {
    details["Started at"] = new Date(context.execution.createdAt).toLocaleString();
  }
  if (context.execution.updatedAt && context.execution.state === "STATE_FINISHED") {
    details["Finished at"] = new Date(context.execution.updatedAt).toLocaleString();
  }

  return details;
}

function props(context: ComponentBaseContext) {
  const base = noopMapper.props(context);
  return {
    ...base,
    metadata: metadataList(context.node),
  };
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  if (!timestamp) return "";
  return renderTimeAgo(new Date(timestamp));
}

// Currently unused, but kept for future per-component expansion (e.g. status panel for deploy).
export type CoolifyExecution = ExecutionInfo;

export const coolifyBaseMapper: ComponentBaseMapper = {
  ...noopMapper,
  props,
  getExecutionDetails,
  subtitle,
};
