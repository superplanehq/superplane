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

    if (data?.name) {
      details["Artifact"] = artifactShortName(data.name as string);
    }

    if (data?.description) {
      details["Description"] = String(data.description);
    }

    if (data?.updateTime) {
      const formatted = formatDateTime(data.updateTime as string);
      if (formatted) details["Updated At"] = formatted;
    }

    if (data?.createTime) {
      const formatted = formatDateTime(data.createTime as string);
      if (formatted) details["Created At"] = formatted;
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
      details["Resource URI"] = String(data.resourceUri);
    }

    const occurrences = data?.occurrences as any[] | null | undefined;
    if (Array.isArray(occurrences)) {
      details["Occurrences"] = String(occurrences.length);
    }

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
