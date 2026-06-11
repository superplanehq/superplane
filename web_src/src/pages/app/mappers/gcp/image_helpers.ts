// Shared types and helpers for the GCP image mappers (create/update/delete image).

export interface ImageNodeMetadata {
  imageName?: string;
}

// imageNameFromValue extracts a display-friendly image name from a configuration
// value, which may be a bare name, a relative/global path, a full selfLink URL,
// or an unresolved expression. Expressions and unrecognized values are returned
// as-is so the collapsed node still shows something meaningful.
export function imageNameFromValue(value: string | undefined): string | undefined {
  if (!value) return undefined;
  const trimmed = value.trim();
  if (!trimmed || trimmed.includes("{{")) return value;
  const match = trimmed.match(/global\/images\/([^/?#]+)/);
  if (match) return match[1];
  if (!trimmed.includes("/")) return trimmed;
  return value;
}
