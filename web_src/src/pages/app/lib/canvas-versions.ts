import type { CanvasesCanvasVersion } from "@/api-client";

export function isPublishedVersion(version: CanvasesCanvasVersion): boolean {
  return version.metadata?.state === "STATE_PUBLISHED";
}

export function isDraftVersion(version: CanvasesCanvasVersion): boolean {
  return version.metadata?.state === "STATE_DRAFT";
}

export function sortPublishedVersionsDesc(versions: CanvasesCanvasVersion[]): CanvasesCanvasVersion[] {
  return versions
    .filter(isPublishedVersion)
    .sort((a, b) => versionSortValue(b.metadata?.publishedAt) - versionSortValue(a.metadata?.publishedAt));
}

export function sortDraftVersionsDesc(versions: CanvasesCanvasVersion[]): CanvasesCanvasVersion[] {
  return versions
    .filter(isDraftVersion)
    .sort(
      (a, b) =>
        versionSortValue(b.metadata?.updatedAt || b.metadata?.createdAt) -
        versionSortValue(a.metadata?.updatedAt || a.metadata?.createdAt),
    );
}

export function formatVersionTimestamp(version?: CanvasesCanvasVersion | null): string | undefined {
  const raw = version?.metadata?.updatedAt || version?.metadata?.createdAt;
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
  if (version?.metadata?.state === "STATE_PUBLISHED") {
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
