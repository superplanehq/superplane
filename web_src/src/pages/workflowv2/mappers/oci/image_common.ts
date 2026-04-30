import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, ExecutionDetailsContext, OutputPayload } from "../types";

export interface OCIImageConfiguration {
  imageId?: string;
  compartmentId?: string;
  displayName?: string;
  instanceId?: string;
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
  const items: MetadataItem[] = [];

  if (config?.displayName) {
    items.push({ icon: "tag", label: config.displayName });
  }

  if (config?.imageId) {
    items.push({ icon: "disc", label: config.imageId });
  }

  if (config?.compartmentId) {
    items.push({ icon: "folder", label: config.compartmentId });
  }

  if (config?.sourceType) {
    items.push({ icon: "hard-drive", label: config.sourceType });
  }

  return items;
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

  if (image.id) details["Image ID"] = image.id;
  if (image.displayName) details["Display Name"] = image.displayName;
  if (image.lifecycleState) details["State"] = image.lifecycleState;
  if (image.operatingSystem) {
    const version = image.operatingSystemVersion ? ` ${image.operatingSystemVersion}` : "";
    details["Operating System"] = `${image.operatingSystem}${version}`;
  }
  if (image.launchMode) details["Launch Mode"] = image.launchMode;
  if (image.timeCreated) details["Created At"] = new Date(image.timeCreated).toLocaleString();

  return details;
}
