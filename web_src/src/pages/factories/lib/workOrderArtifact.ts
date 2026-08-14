export type ArtifactData = Record<string, unknown> | undefined;

/** GitHub-style pull request lifecycle states a PR artifact's chip can render. */
export type PrArtifactState = "open" | "draft" | "closed" | "merged";

const PR_ARTIFACT_STATES: readonly PrArtifactState[] = ["open", "draft", "closed", "merged"];

// Keys we accept for the PR number in the free-form artifact data map.
// The Add Work Order Artifact component's example uses `number`; some
// authors reach for `prNumber`. Tolerate both so links render as `#1234`
// regardless of which convention was used.
const PR_NUMBER_KEYS = ["number", "prNumber"] as const;

/**
 * Returns `#<n>` when the free-form artifact data carries a PR number,
 * otherwise undefined so callers can fall back to title / URL / a generic
 * label. Any leading `#` is stripped to avoid rendering `##`.
 */
export function formatPrArtifactLabel(data: ArtifactData): string | undefined {
  for (const key of PR_NUMBER_KEYS) {
    const raw = extractArtifactField(data, key);
    if (raw === undefined) {
      continue;
    }

    const digits = raw.replace(/^#/, "").trim();
    if (digits) {
      return `#${digits}`;
    }
  }
  return undefined;
}

export function extractArtifactMarkdownBody(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "body");
}

export function extractArtifactUrl(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "url");
}

export function extractArtifactTitle(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "title");
}

export function extractArtifactName(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "name");
}

/**
 * Returns the PR's lifecycle state when the free-form artifact data
 * carries a known value, normalizing case. Returns undefined for
 * missing/unrecognized values (e.g. artifacts attached before this
 * field existed) so callers can fall back to the default "open" look
 * instead of crashing or rendering something misleading.
 */
export function extractPrArtifactState(data: ArtifactData): PrArtifactState | undefined {
  const raw = extractArtifactField(data, "state");
  if (!raw) {
    return undefined;
  }

  const normalized = raw.toLowerCase();
  return PR_ARTIFACT_STATES.find((state) => state === normalized);
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
