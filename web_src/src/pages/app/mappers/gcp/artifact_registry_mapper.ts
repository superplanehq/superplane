import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  SubtitleContext,
} from "../types";
import { baseMapper } from "./base";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getArtifactOutputPayload, getArtifactData, artifactShortName } from "./artifact_registry";
import gcpArtifactRegistryIcon from "@/assets/icons/integrations/gcp.artifactregistry.svg";

export const getArtifactMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      iconSrc: gcpArtifactRegistryIcon,
      metadata: artifactActionMetadataList(context.node),
    };
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
      if (formatted) details["Image Created At"] = formatted;
    }

    if (data?.updateTime) {
      const formatted = formatDateTime(data.updateTime as string);
      if (formatted) details["Image Updated At"] = formatted;
    }

    const sizeBytes = data?.metadata?.imageSizeBytes;
    if (sizeBytes) {
      details["Size"] = formatBytes(Number(sizeBytes));
    }

    const digest = artifactShortName(data?.name as string | undefined);
    if (digest) {
      details["Digest"] = digest;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

export const getArtifactAnalysisMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      ...baseMapper.props(context),
      iconSrc: gcpArtifactRegistryIcon,
      metadata: artifactActionMetadataList(context.node),
    };
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

    if (typeof data?.critical === "number" && data.critical > 0) {
      details["Critical"] = String(data.critical);
    }
    if (typeof data?.high === "number" && data.high > 0) {
      details["High"] = String(data.high);
    }
    if (typeof data?.fixAvailable === "number" && data.fixAvailable > 0) {
      details["Fixes Available"] = String(data.fixAvailable);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
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

function artifactActionMetadataList(node: NodeInfo): MetadataItem[] {
  const config = (node.configuration as Record<string, any> | undefined) ?? {};
  const inputMode = String(config.inputMode || "url").toLowerCase();
  const metadata: MetadataItem[] = [];

  if (inputMode === "select") {
    metadata.push({ icon: "funnel", label: "Select from Registry" });

    const scope = [config.location, config.repository, config.package].filter(Boolean).map(String).join(" / ");
    if (scope) {
      metadata.push({ icon: "package", label: scope });
    }

    if (config.version) {
      metadata.push({ icon: "tag", label: String(config.version) });
    }

    return metadata;
  }

  metadata.push({ icon: "link", label: "Resource URL" });
  if (config.resourceUrl) {
    metadata.push({ icon: "package", label: compactValue(String(config.resourceUrl), 72) });
  }
  return metadata;
}

function compactValue(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }

  return `${value.slice(0, maxLength)}...`;
}
