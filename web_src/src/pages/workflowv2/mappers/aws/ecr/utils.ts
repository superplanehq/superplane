import { MetadataItem } from "@/ui/metadataList";
import { EcrTriggerConfiguration, EcrTriggerMetadata } from "./types";

export function getRepositoryLabel(
  metadata?: EcrTriggerMetadata,
  configuration?: EcrTriggerConfiguration,
  fallback?: string,
): string | undefined {
  const repositoryRef =
    metadata?.repository?.repositoryName ||
    metadata?.repository?.repositoryArn ||
    configuration?.repository ||
    fallback;

  if (!repositoryRef) {
    return undefined;
  }

  return extractRepositoryName(repositoryRef);
}

export function extractRepositoryName(repositoryRef: string): string {
  const trimmed = repositoryRef.trim();
  if (!trimmed.startsWith("arn:")) {
    return trimmed;
  }

  const parts = trimmed.split("repository/");
  if (parts.length !== 2 || !parts[1]) {
    return trimmed;
  }

  return parts[1];
}

export function formatTags(tags?: string[]): string {
  if (!tags || tags.length === 0) {
    return "-";
  }

  return tags.join(", ");
}

export function formatTagLabel(tags?: string[]): string | undefined {
  if (!tags || tags.length === 0) {
    return undefined;
  }

  if (tags.length === 1) {
    return tags[0];
  }

  return `${tags[0]} +${tags.length - 1}`;
}

export function buildRepositoryMetadataItems(
  metadata?: EcrTriggerMetadata,
  configuration?: EcrTriggerConfiguration,
): MetadataItem[] {
  const repositoryLabel = getRepositoryLabel(metadata, configuration);

  if (!repositoryLabel) {
    return [];
  }

  return [
    {
      icon: "package",
      label: repositoryLabel,
    },
  ];
}

export function numberOrZero(value?: number): number {
  if (value === undefined || value === null) {
    return 0;
  }

  return value;
}

export function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
