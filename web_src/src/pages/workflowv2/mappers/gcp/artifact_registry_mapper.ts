import { ComponentBaseProps } from "@/ui/componentBase";
import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "../types";
import { baseMapper } from "./base";
import { formatTimeAgo } from "@/utils/date";
import { getArtifactOutputPayload, getArtifactData, artifactShortName } from "./artifact_registry";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";

export const getArtifactMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return { ...baseMapper.props(context), iconSrc: gcpArtifactRegistryIcon };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = getArtifactOutputPayload(context.execution);
    const data = getArtifactData(context.execution) as Record<string, any> | undefined;
    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Retrieved At"] = new Date(payload.timestamp).toLocaleString();
    }

    const dockerUri = buildDockerUri(data?.metadata?.name as string | undefined);
    if (dockerUri) {
      details["Image"] = dockerUri;
    }

    if (data?.createTime) {
      const formatted = formatDateTime(data.createTime as string);
      if (formatted) details["Created"] = formatted;
    }

    if (data?.updateTime) {
      const formatted = formatDateTime(data.updateTime as string);
      if (formatted) details["Updated"] = formatted;
    }

    const sizeBytes = data?.metadata?.imageSizeBytes;
    if (sizeBytes) {
      details["Size"] = formatBytes(Number(sizeBytes));
    }

    const digest = artifactShortName(data?.name as string | undefined);
    if (digest) {
      details["Digest"] = digest;
    }

    if (data?.metadata?.mediaType) {
      details["Media Type"] = String(data.metadata.mediaType);
    }

    if (data?.metadata?.buildTime) {
      const formatted = formatDateTime(data.metadata.buildTime as string);
      if (formatted) details["Build Time"] = formatted;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

export const getArtifactAnalysisMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return { ...baseMapper.props(context), iconSrc: gcpArtifactRegistryIcon };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = getArtifactOutputPayload(context.execution);
    const data = getArtifactData(context.execution) as Record<string, any> | undefined;
    const details: Record<string, string> = {};

    if (payload?.timestamp) {
      details["Retrieved At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (data?.resourceUri) {
      details["Image"] = String(data.resourceUri);
    }

    if (data?.scanStatus) {
      details["Scan Status"] = String(data.scanStatus);
    }

    if (typeof data?.vulnerabilities === "number") {
      details["Vulnerabilities"] = String(data.vulnerabilities);
    }

    if (typeof data?.critical === "number" && data.critical > 0) details["Critical"] = String(data.critical);
    if (typeof data?.high === "number" && data.high > 0) details["High"] = String(data.high);
    if (typeof data?.medium === "number" && data.medium > 0) details["Medium"] = String(data.medium);
    if (typeof data?.fixAvailable === "number" && data.fixAvailable > 0)
      details["Fix Available"] = String(data.fixAvailable);

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

function formatDateTime(value?: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString();
}

// Converts a metadata.name like
// "projects/P/locations/L/repositories/R/dockerImages/IMG@sha256:DIGEST"
// into "L-docker.pkg.dev/P/R/IMG@sha256:DIGEST"
function buildDockerUri(metadataName?: string): string | undefined {
  if (!metadataName) return undefined;
  const m = metadataName.match(/^projects\/([^/]+)\/locations\/([^/]+)\/repositories\/([^/]+)\/dockerImages\/(.+)$/);
  if (!m) return undefined;
  const [, project, location, repo, imageRef] = m;
  return `https://${location}-docker.pkg.dev/${project}/${repo}/${imageRef}`;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
