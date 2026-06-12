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

type CloudSQLOutputs = {
  default?: Array<{
    data?: {
      name?: string;
      instance?: string;
      charset?: string;
      collation?: string;
      deleted?: boolean;
    };
  }>;
};

function formatLocalDateTime(value?: string): string | undefined {
  return value ? new Date(value).toLocaleString() : undefined;
}

function databaseDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  const completedAt = formatLocalDateTime(context.execution.updatedAt || context.execution.createdAt);
  if (completedAt) details["Completed At"] = completedAt;

  const item = (context.execution.outputs as CloudSQLOutputs | undefined)?.default?.[0]?.data;
  if (!item) return details;
  if (item.name) details["Database"] = item.name;
  if (item.instance) details["Instance"] = item.instance;
  if (item.charset) details["Charset"] = item.charset;
  if (item.collation) details["Collation"] = item.collation;
  if (item.deleted) details["Deleted"] = "true";
  return details;
}

// displayValue returns a trimmed string only when it is safe to show on the
// node: empty values and unresolved expressions (e.g. "{{ $.x }}") are hidden,
// matching the other GCP mappers.
function displayValue(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined;
  const trimmed = String(value).trim();
  if (!trimmed || trimmed.includes("{{")) return undefined;
  return trimmed;
}

function cloudsqlMetadataList(node: NodeInfo): MetadataItem[] {
  const config = (node.configuration as Record<string, unknown> | undefined) ?? {};
  const metadata: MetadataItem[] = [];
  const instance = displayValue(config.instance);
  if (instance) metadata.push({ icon: "server", label: instance });
  const db = displayValue(config.name ?? config.database);
  if (db) metadata.push({ icon: "database", label: db });
  return metadata;
}

export const createDatabaseMapper: ComponentBaseMapper = {
  props: cloudsqlProps,
  getExecutionDetails: databaseDetails,
  subtitle: cloudsqlSubtitle,
};

export const getDatabaseMapper: ComponentBaseMapper = {
  props: cloudsqlProps,
  getExecutionDetails: databaseDetails,
  subtitle: cloudsqlSubtitle,
};

export const deleteDatabaseMapper: ComponentBaseMapper = {
  props: cloudsqlProps,
  getExecutionDetails: databaseDetails,
  subtitle: cloudsqlSubtitle,
};

// Per-action success labels so the node badge says what the component did.
export const CLOUDSQL_CREATED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("created");
export const CLOUDSQL_FETCHED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("fetched");
export const CLOUDSQL_DELETED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("deleted");
