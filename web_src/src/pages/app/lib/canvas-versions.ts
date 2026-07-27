import type { CanvasesCanvasVersion, CanvasesCanvasVersionMetadata } from "@/api-client";

export type CanvasVersionListItem = CanvasesCanvasVersionMetadata;

export function canvasVersionId(version?: CanvasesCanvasVersion | CanvasVersionListItem | null): string | undefined {
  if (!version) {
    return undefined;
  }

  if ("metadata" in version) {
    return version.metadata?.id;
  }

  return (version as CanvasVersionListItem).id;
}

export function canvasVersionShell(metadata?: CanvasVersionListItem | null): CanvasesCanvasVersion | null {
  if (!metadata?.id) {
    return null;
  }

  return { metadata };
}

export function toCanvasVersionShell(
  version?: CanvasesCanvasVersion | CanvasVersionListItem,
): CanvasesCanvasVersion | undefined {
  if (!version) {
    return undefined;
  }

  if ("metadata" in version && version.metadata) {
    return version;
  }

  return canvasVersionShell(version as CanvasVersionListItem) ?? undefined;
}

export function sortVersionsDesc(versions: CanvasVersionListItem[]): CanvasVersionListItem[] {
  return [...versions].sort(
    (a, b) => versionSortValue(b.updatedAt || b.createdAt) - versionSortValue(a.updatedAt || a.createdAt),
  );
}

/** @deprecated All versions are main-branch commits; use sortVersionsDesc. */
export function sortPublishedVersionsDesc(versions: CanvasVersionListItem[]): CanvasVersionListItem[] {
  return sortVersionsDesc(versions);
}

export function formatVersionTimestamp(version?: CanvasVersionListItem | null): string | undefined {
  const raw = version?.updatedAt || version?.createdAt;
  if (!raw) {
    return undefined;
  }

  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function formatLegacyVersionLabel(version?: CanvasVersionListItem | null): string | undefined {
  const raw = version?.createdAt;
  if (!raw) {
    return undefined;
  }

  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  const formatted = date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
  return `Update from ${formatted}`;
}

export function formatVersionLabel(version?: CanvasVersionListItem | null): string {
  const message = version?.commitMessage?.trim();
  if (message) {
    return message;
  }

  return formatLegacyVersionLabel(version) ?? "Untitled update";
}

export function formatVersionLabelWithTimestamp(version?: CanvasVersionListItem | null): string {
  const message = version?.commitMessage?.trim();
  if (message) {
    const timestamp = formatVersionTimestamp(version);
    if (!timestamp) {
      return message;
    }

    return `${message} · ${timestamp}`;
  }

  return formatLegacyVersionLabel(version) ?? "Untitled update";
}

export function versionSortValue(raw?: string): number {
  if (!raw) {
    return 0;
  }

  const parsed = Date.parse(raw);
  return Number.isNaN(parsed) ? 0 : parsed;
}
