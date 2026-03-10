import { ExecutionInfo, OutputPayload, StateFunction } from "../types";
import { DEFAULT_EVENT_STATE_MAP, EventState } from "@/ui/componentBase";
import { EventStateRegistry } from "../types";

export type ArtifactPushData = {
  action?: string;
  digest?: string;
  tag?: string;
};

export type VulnerabilityPackageIssue = {
  affectedPackage?: string;
  affectedVersion?: { name?: string; kind?: string };
  fixedVersion?: { name?: string; kind?: string };
};

export type VulnerabilityData = {
  severity?: string;
  cvssScore?: number;
  packageIssue?: VulnerabilityPackageIssue[];
};

export type VulnerabilityFinding = {
  name?: string;
  vulnerability?: VulnerabilityData;
};

export type OccurrenceData = {
  name?: string;
  resourceUri?: string;
  noteName?: string;
  kind?: string;
  vulnerability?: VulnerabilityData;
};

export type GetArtifactAnalysisData = {
  resourceUri?: string;
  scanStatus?: string;
  vulnerabilities?: number;
  critical?: number;
  high?: number;
  medium?: number;
  low?: number;
  fixAvailable?: number;
};

export type ArtifactVersionFingerprint = {
  type?: string;
  value?: string;
};

export type ArtifactVersionMetadata = {
  buildTime?: string;
  imageSizeBytes?: string;
  mediaType?: string;
  name?: string;
};

export type ArtifactVersionData = {
  name?: string;
  createTime?: string;
  updateTime?: string;
  description?: string;
  fingerprints?: ArtifactVersionFingerprint[];
  metadata?: ArtifactVersionMetadata;
};

type ArtifactOutputPayload = OutputPayload & {
  data?: Record<string, any>;
};

export function getArtifactOutputPayload(execution: ExecutionInfo): ArtifactOutputPayload | undefined {
  const outputs = execution.outputs as
    | { passed?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
    | undefined;
  const payload = outputs?.passed?.[0] ?? outputs?.failed?.[0] ?? outputs?.default?.[0];
  if (!payload || typeof payload !== "object") {
    return undefined;
  }
  return payload as ArtifactOutputPayload;
}

export function getArtifactData(execution: ExecutionInfo): Record<string, any> | undefined {
  const payload = getArtifactOutputPayload(execution);
  return payload?.data as Record<string, any> | undefined;
}

export function buildArtifactSummaryDetails({ timestamp }: { timestamp?: string }): Record<string, string> {
  const details: Record<string, string> = {};

  if (timestamp) {
    const formatted = formatDateTime(timestamp);
    if (formatted) details["Executed At"] = formatted;
  }

  return details;
}

export const artifactRegistryExecutionStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) {
    return "neutral";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "success";
  }

  return "failed";
};

export const ARTIFACT_REGISTRY_EXECUTION_STATE_REGISTRY: EventStateRegistry = {
  stateMap: DEFAULT_EVENT_STATE_MAP,
  getState: artifactRegistryExecutionStateFunction,
};

function formatDateTime(value?: string): string | undefined {
  if (!value) {
    return undefined;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }
  return date.toLocaleString();
}

export function artifactShortName(name?: string): string {
  if (!name) return "";
  const parts = name.split("/");
  return parts[parts.length - 1] ?? name;
}
