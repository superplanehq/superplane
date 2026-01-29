import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { EcrTriggerConfiguration, EcrTriggerMetadata } from "./types";

const SEVERITY_ORDER = ["CRITICAL", "HIGH", "MEDIUM", "LOW", "INFORMATIONAL", "UNDEFINED"];

export function buildSubtitle(contentParts: Array<string | undefined>, createdAt?: string): string {
  const content = contentParts.filter(Boolean).join(" · ");
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";

  if (content && timeAgo) {
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}

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

export function formatSeverityCounts(counts?: Record<string, number>): string {
  if (!counts || Object.keys(counts).length === 0) {
    return "-";
  }

  const entries = Object.entries(counts);
  const orderedEntries = entries.sort((a, b) => {
    const aIndex = SEVERITY_ORDER.indexOf(a[0]);
    const bIndex = SEVERITY_ORDER.indexOf(b[0]);

    if (aIndex === -1 && bIndex === -1) {
      return a[0].localeCompare(b[0]);
    }

    if (aIndex === -1) return 1;
    if (bIndex === -1) return -1;

    return aIndex - bIndex;
  });

  return orderedEntries.map(([severity, count]) => `${severity}: ${count}`).join(", ");
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

export function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
