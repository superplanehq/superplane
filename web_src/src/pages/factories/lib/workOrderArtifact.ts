export type ArtifactData = Record<string, unknown> | undefined;

/**
 * Prefer `data.prNumber` for PR artifact labels so links read as `#1234`,
 * matching how git hosts render them. Returns undefined when the field is
 * missing or empty, letting callers fall back to title / URL / a generic
 * label.
 */
export function formatPrArtifactLabel(data: ArtifactData): string | undefined {
  const raw = extractArtifactField(data, "prNumber");
  if (raw === undefined) {
    return undefined;
  }

  const digits = raw.replace(/^#/, "").trim();
  return digits ? `#${digits}` : undefined;
}

export function extractArtifactMarkdownBody(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "body");
}

function extractArtifactField(data: ArtifactData, key: string): string | undefined {
  if (!data) {
    return undefined;
  }

  const value = data[key];
  if (typeof value === "string") {
    return value.trim() !== "" ? value : undefined;
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }
  return undefined;
}
