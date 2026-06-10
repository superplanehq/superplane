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
import gcpIcon from "@/assets/icons/integrations/gcp.svg";
import type { MetadataItem } from "@/ui/metadataList";

function cloudsqlProps(context: ComponentBaseContext): ComponentBaseProps {
  return {
    ...baseMapper.props(context),
    iconSrc: gcpIcon,
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

function cloudsqlMetadataList(node: NodeInfo): MetadataItem[] {
  const config = (node.configuration as Record<string, unknown> | undefined) ?? {};
  const metadata: MetadataItem[] = [];
  if (config.instance) metadata.push({ icon: "server", label: String(config.instance) });
  const db = config.name || config.database;
  if (db) metadata.push({ icon: "database", label: String(db) });
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

export const CLOUDSQL_ACTION_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("completed");
