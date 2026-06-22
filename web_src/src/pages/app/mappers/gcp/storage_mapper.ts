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
import storageIcon from "@/assets/icons/integrations/gcp.storage.svg";
import type { MetadataItem } from "@/ui/metadataList";

function storageSubtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

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

type BucketOutputs = {
  default?: Array<{
    data?: {
      name?: string;
      location?: string;
      locationType?: string;
      storageClass?: string;
      consoleUrl?: string;
      deleted?: boolean;
    };
  }>;
};

// bucketDetails shows the timestamp first, then at most a handful of the most
// useful fields, ending with the Console link so the user can jump to the bucket.
function bucketDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  const completedAt = formatLocalDateTime(context.execution.updatedAt || context.execution.createdAt);
  if (completedAt) details["Completed At"] = completedAt;

  const item = (context.execution.outputs as BucketOutputs | undefined)?.default?.[0]?.data;
  if (!item) return details;

  // Delete emits a small confirmation payload.
  if (item.deleted) {
    if (item.name) details["Bucket"] = item.name;
    details["Deleted"] = "true";
    return details;
  }

  if (item.name) details["Bucket"] = item.name;
  if (item.location) details["Location"] = item.location;
  if (item.storageClass) details["Storage Class"] = item.storageClass;
  if (item.consoleUrl) details["Console"] = item.consoleUrl;
  return details;
}

function bucketMetadataList(node: NodeInfo): MetadataItem[] {
  const config = (node.configuration as Record<string, unknown> | undefined) ?? {};
  const metadata: MetadataItem[] = [];
  // create uses "name"; get/delete use "bucket".
  const bucket = displayValue(config.name ?? config.bucket);
  if (bucket) metadata.push({ icon: "database", label: bucket });
  const location = displayValue(config.location);
  if (location) metadata.push({ icon: "globe", label: location });
  const storageClass = displayValue(config.storageClass);
  if (storageClass) metadata.push({ icon: "tag", label: storageClass });
  return metadata;
}

function bucketProps(context: ComponentBaseContext): ComponentBaseProps {
  return {
    ...baseMapper.props(context),
    iconSrc: storageIcon,
    metadata: bucketMetadataList(context.node),
  };
}

export const createBucketMapper: ComponentBaseMapper = {
  props: bucketProps,
  getExecutionDetails: bucketDetails,
  subtitle: storageSubtitle,
};

export const getBucketMapper: ComponentBaseMapper = {
  props: bucketProps,
  getExecutionDetails: bucketDetails,
  subtitle: storageSubtitle,
};

export const deleteBucketMapper: ComponentBaseMapper = {
  props: bucketProps,
  getExecutionDetails: bucketDetails,
  subtitle: storageSubtitle,
};

// Per-action success labels so the node badge says what the component did.
export const STORAGE_CREATED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("created");
export const STORAGE_FETCHED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("fetched");
export const STORAGE_DELETED_STATE_REGISTRY: EventStateRegistry = buildActionStateRegistry("deleted");
