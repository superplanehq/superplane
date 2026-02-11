import { MetadataItem } from "@/ui/metadataList";
import { CodeBuildConfiguration, CodeBuildTriggerMetadata } from "./types";

export function getProjectLabel(
  metadata?: CodeBuildTriggerMetadata,
  configuration?: CodeBuildConfiguration,
  fallback?: string,
): string | undefined {
  const projectRef =
    metadata?.project?.projectName || metadata?.project?.projectArn || configuration?.project || fallback;

  if (!projectRef) {
    return undefined;
  }

  return extractProjectName(projectRef);
}

export function extractProjectName(projectRef: string): string {
  const trimmed = projectRef.trim();
  if (!trimmed.startsWith("arn:")) {
    return trimmed;
  }

  const parts = trimmed.split("project/");
  if (parts.length !== 2 || !parts[1]) {
    return trimmed;
  }

  return parts[1];
}

export function buildProjectMetadataItems(
  metadata?: CodeBuildTriggerMetadata,
  configuration?: CodeBuildConfiguration,
  fallback?: string,
): MetadataItem[] {
  const projectLabel = getProjectLabel(metadata, configuration, fallback);
  if (!projectLabel) {
    return [];
  }

  return [
    {
      icon: "hammer",
      label: projectLabel,
    },
  ];
}
