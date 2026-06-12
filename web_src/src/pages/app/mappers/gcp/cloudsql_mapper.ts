import type React from "react";
import type {
  ComponentBaseMapper,
  ComponentBaseContext,
  EventStateRegistry,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import type { ComponentBaseProps } from "@/ui/componentBase";
import { baseMapper } from "./base";
import { buildActionStateRegistry } from "../utils";
import { renderTimeAgo } from "@/components/TimeAgo";
import cloudSqlIcon from "@/assets/icons/integrations/gcp.cloudsql.svg";
import type { MetadataItem } from "@/ui/metadataList";

function cloudsqlProps(context: ComponentBaseContext): ComponentBaseProps {
  return {
    ...baseMapper.props(context),
    iconSrc: cloudSqlIcon,
    metadata: cloudsqlMetadataList(context.node),
  };
}

function cloudsqlSubtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

type InstanceOutputs = {
  default?: Array<{
    data?: {
      name?: string;
      state?: string;
      databaseVersion?: string;
      connectionName?: string;
      ipAddress?: string;
      deleted?: boolean;
    };
  }>;
};

function formatLocalDateTime(value?: string): string | undefined {
  return value ? new Date(value).toLocaleString() : undefined;
}

// displayValue hides empty values and unresolved expressions (e.g. "{{ $.x }}"),
// matching the other GCP mappers.
function displayValue(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined;
  const trimmed = String(value).trim();
  if (!trimmed || trimmed.includes("{{")) return undefined;
  return trimmed;
}

function instanceDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  // Timestamp first, then at most a handful of the most useful fields.
  const completedAt = formatLocalDateTime(context.execution.updatedAt || context.execution.createdAt);
  if (completedAt) details["Completed At"] = completedAt;

  const item = (context.execution.outputs as InstanceOutputs | undefined)?.default?.[0]?.data;
  if (!item) return details;

  // Delete emits a small confirmation payload.
  if (item.deleted) {
    if (item.name) details["Instance"] = item.name;
    details["Deleted"] = "true";
    return details;
  }

  if (item.state) details["State"] = item.state;
  if (item.databaseVersion) details["Version"] = item.databaseVersion;
  if (item.connectionName) details["Connection"] = item.connectionName;
  if (item.ipAddress) details["IP Address"] = item.ipAddress;
  return details;
}

function cloudsqlMetadataList(node: NodeInfo): MetadataItem[] {
  const config = (node.configuration as Record<string, unknown> | undefined) ?? {};
  const metadata: MetadataItem[] = [];
  // create uses "name"; get/delete use "instance".
  const instance = displayValue(config.name ?? config.instance);
  if (instance) metadata.push({ icon: "database", label: instance });
  const version = displayValue(config.databaseVersion);
  if (version) metadata.push({ icon: "tag", label: version });
  const region = displayValue(config.region);
  if (region) metadata.push({ icon: "globe", label: region });
  return metadata;
}

export const createInstanceMapper: ComponentBaseMapper = {
  props: cloudsqlProps,
  getExecutionDetails: instanceDetails,
  subtitle: cloudsqlSubtitle,
};

export const getInstanceMapper: ComponentBaseMapper = {
  props: cloudsqlProps,
  getExecutionDetails: instanceDetails,
  subtitle: cloudsqlSubtitle,
};

export const deleteInstanceMapper: ComponentBaseMapper = {
  props: cloudsqlProps,
  getExecutionDetails: instanceDetails,
  subtitle: cloudsqlSubtitle,
};

// Per-action success labels so the node badge says what the component did.
export const CLOUDSQL_CREATED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("created");
export const CLOUDSQL_FETCHED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("fetched");
export const CLOUDSQL_DELETED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("deleted");
