import type { CanvasesCanvasVersion } from "@/api-client";

export function sortVersionsDesc(versions: CanvasesCanvasVersion[]): CanvasesCanvasVersion[] {
  return [...versions].sort(
    (a, b) =>
      versionSortValue(b.metadata?.updatedAt || b.metadata?.createdAt) -
      versionSortValue(a.metadata?.updatedAt || a.metadata?.createdAt),
  );
}

/** @deprecated All versions are main-branch commits; use sortVersionsDesc. */
export function sortPublishedVersionsDesc(versions: CanvasesCanvasVersion[]): CanvasesCanvasVersion[] {
  return sortVersionsDesc(versions);
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

function formatLegacyVersionLabel(version?: CanvasesCanvasVersion | null): string | undefined {
  const raw = version?.metadata?.createdAt;
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

export function formatVersionLabel(version?: CanvasesCanvasVersion | null): string {
  const message = version?.metadata?.commitMessage?.trim();
  if (message) {
    return message;
  }

  return formatLegacyVersionLabel(version) ?? "Untitled update";
}

export function formatVersionLabelWithTimestamp(version?: CanvasesCanvasVersion | null): string {
  const message = version?.metadata?.commitMessage?.trim();
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
