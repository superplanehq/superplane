import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, ExecutionDetailsContext, OutputPayload } from "../types";

export const MAX_NODE_METADATA_ITEMS = 3;

export interface OCIImageConfiguration {
  image?: string;
  compartment?: string;
  displayName?: string;
  instance?: string;
  instanceId?: string;
  sourceType?: string;
  bucket?: string;
  object?: string;
}

interface OCIImageNodeMetadata {
  imageName?: string;
  compartmentName?: string;
  instanceName?: string;
  displayName?: string;
  sourceType?: string;
}

export interface OCIImage {
  id?: string;
  displayName?: string;
  lifecycleState?: string;
  compartmentId?: string;
  operatingSystem?: string;
  operatingSystemVersion?: string;
  launchMode?: string;
  sizeInMBs?: number;
  timeCreated?: string;
  freeformTags?: Record<string, string>;
}

export interface OCIImageOutputData {
  image?: OCIImage;
  imageId?: string;
  state?: string;
  deletedAt?: string;
}

type OCIImageOutputPayload = OutputPayload & {
  data?: OCIImageOutputData;
};

interface OCIImageExecutionMetadata {
  startedAt?: string;
}

export function getExecutedAt(context: ExecutionDetailsContext): string | undefined {
  const metadata = context.execution.metadata as OCIImageExecutionMetadata | undefined;
  const ts = metadata?.startedAt ?? context.execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

export function getOutputData(context: ExecutionDetailsContext): OCIImageOutputData | undefined {
  const outputs = context.execution.outputs as { default?: OCIImageOutputPayload[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as OCIImageOutputData;
}

export function imageMetadataList(node: ComponentBaseContext["node"]): MetadataItem[] {
  const config = node.configuration as OCIImageConfiguration | undefined;
  const nodeMetadata = node.metadata as OCIImageNodeMetadata | undefined;

  const displayName = nodeMetadata?.displayName ?? config?.displayName;
  const sourceType = sourceTypeLabel(nodeMetadata?.sourceType ?? config?.sourceType);

  return [
    metadataItem("tag", displayName),
    metadataItem("disc", nodeMetadata?.imageName),
    metadataItem("folder", nodeMetadata?.compartmentName),
    metadataItem("server", nodeMetadata?.instanceName),
    metadataItem("archive", config?.bucket),
    metadataItem("file", config?.object),
    metadataItem("hard-drive", sourceType),
  ]
    .filter(isMetadataItem)
    .slice(0, MAX_NODE_METADATA_ITEMS);
}

export function metadataItem(icon: MetadataItem["icon"], label?: string): MetadataItem | undefined {
  return label ? { icon, label } : undefined;
}

export function isMetadataItem(item: MetadataItem | undefined): item is MetadataItem {
  return item !== undefined;
}

function sourceTypeLabel(sourceType?: string): string | undefined {
  switch (sourceType) {
    case "instance":
      return "Instance";
    case "objectStorageUri":
      return "Object Storage URL";
    case "objectStorageObject":
      return "Object Storage Object";
    default:
      return sourceType;
  }
}

export function addExecutedAt(details: Record<string, string>, context: ExecutionDetailsContext): void {
  const executedAt = getExecutedAt(context);
  if (executedAt) {
    details["Executed At"] = executedAt;
  }
}

export function imageDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  addExecutedAt(details, context);

  const data = getOutputData(context);
  const image = data?.image;
  if (!image) return details;

  if (image.displayName) details["Display Name"] = image.displayName;
  if (image.lifecycleState) details["State"] = image.lifecycleState;
  if (image.operatingSystem) {
    const version = image.operatingSystemVersion ? ` ${image.operatingSystemVersion}` : "";
    details["Operating System"] = `${image.operatingSystem}${version}`;
  }
  if (image.launchMode) details["Launch Mode"] = image.launchMode;
  if (image.freeformTags && Object.keys(image.freeformTags).length > 0) {
    details["Tags"] = Object.entries(image.freeformTags)
      .map(([key, value]) => `${key}: ${value}`)
      .join(", ");
  }

  return details;
}
