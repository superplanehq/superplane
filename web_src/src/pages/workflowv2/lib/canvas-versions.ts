import type { CanvasesCanvasVersion } from "@/api-client";

export function formatVersionTimestamp(version?: CanvasesCanvasVersion | null): string | undefined {
  const raw = version?.metadata?.updatedAt || version?.metadata?.publishedAt || version?.metadata?.createdAt;
  if (!raw) {
    return undefined;
  }

  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

export function formatVersionLabel(version?: CanvasesCanvasVersion | null): string {
  if (version?.metadata?.isPublished) {
    return "Published version";
  }

  return "Draft version";
}

export function formatVersionLabelWithTimestamp(version?: CanvasesCanvasVersion | null): string {
  const label = formatVersionLabel(version);
  const timestamp = formatVersionTimestamp(version);
  if (!timestamp) {
    return label;
  }

  return `${label} · ${timestamp}`;
}

export function versionSortValue(raw?: string): number {
  if (!raw) {
    return 0;
  }

  const parsed = Date.parse(raw);
  return Number.isNaN(parsed) ? 0 : parsed;
}
