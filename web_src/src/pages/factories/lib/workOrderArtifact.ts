import { FileImage, FileText, FileVideo } from "lucide-react";

export type ArtifactData = Record<string, unknown> | undefined;

/** Image, video, or document icon. Same choice the artifact row already uses. */
export function fileArtifactIcon(contentType?: string): typeof FileText {
  if (contentType?.startsWith("image/")) {
    return FileImage;
  }
  if (contentType?.startsWith("video/")) {
    return FileVideo;
  }
  return FileText;
}

/** GitHub-style pull request lifecycle states a PR artifact's chip can render. */
export type PrArtifactState = "open" | "draft" | "closed" | "merged";

const PR_ARTIFACT_STATES: readonly PrArtifactState[] = ["open", "draft", "closed", "merged"];

// Keys we accept for the PR number in the free-form artifact data map.
// The Add Task Artifact component's example uses `number`; some
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

/**
 * Returns the artifact's outgoing link, if any. `url` is SuperPlane's
 * canonical key; `html_url` is GitHub's own field name and is what
 * `github.createPullRequest` writes into free-form data, so it lands
 * here without the caller having to remap.
 */
export function extractArtifactUrl(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "url") ?? extractArtifactField(data, "html_url");
}

/**
 * Browse URL for a branch artifact that carries no `url`. The Add Task
 * Artifact component writes a tree URL at attach time, so this only covers
 * branches attached before that: they hold `repository` (`owner/repo` or a
 * repository http(s) URL) plus the branch `name`.
 */
export function branchTreeUrl(data: ArtifactData): string | undefined {
  const repository = extractArtifactField(data, "repository") ?? extractArtifactField(data, "repo");
  const branch = extractArtifactName(data)?.trim();
  if (!repository || !branch) {
    return undefined;
  }

  const path = branch.split("/").map(encodeURIComponent).join("/");
  const repo = repository.trim().replace(/\/+$/, "");
  if (/^https?:\/\//i.test(repo)) {
    return treeUrlFromRepositoryUrl(repo, path);
  }
  return /^[^/\s]+\/[^/\s]+$/.test(repo) ? `https://github.com/${repo}/tree/${path}` : undefined;
}

export function extractArtifactTitle(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "title");
}

export function extractArtifactName(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "name");
}

export function extractArtifactFilename(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "filename");
}

export function extractArtifactContentType(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "contentType");
}

/** Byte count from `sizeBytes`, or a numeric `size`. */
export function extractArtifactSizeBytes(data: ArtifactData): number | undefined {
  if (!data) {
    return undefined;
  }
  return readPositiveBytes(data.sizeBytes) ?? readPositiveBytes(data.size);
}

/** Compact size for a file row badge: `348 B`, `12 KB`, `1.2 MB`. */
export function formatArtifactSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  const kb = bytes / 1024;
  if (kb < 1024) {
    return `${formatSizeNumber(kb)} KB`;
  }
  const mb = kb / 1024;
  if (mb < 1024) {
    return `${formatSizeNumber(mb)} MB`;
  }
  return `${formatSizeNumber(mb / 1024)} GB`;
}

/**
 * Label for the artifact size badge. Prefers stored bytes, then an already
 * formatted `size` string, then the markdown body length.
 */
export function artifactSizeLabel(data: ArtifactData): string | undefined {
  const bytes = extractArtifactSizeBytes(data);
  if (bytes !== undefined) {
    return formatArtifactSize(bytes);
  }
  if (typeof data?.size === "string" && /[A-Za-z]/.test(data.size)) {
    const label = data.size.trim();
    return label !== "" ? label : undefined;
  }
  const body = extractArtifactMarkdownBody(data);
  if (body) {
    return formatArtifactSize(new TextEncoder().encode(body).length);
  }
  return undefined;
}

/**
 * Returns the PR's lifecycle state as SuperPlane sees it, using both
 * the canonical `state` field and GitHub-native `merged` / `draft`
 * flags so a chip renders correctly even when nobody rewrote the raw
 * webhook payload into SuperPlane's vocabulary.
 *
 * Precedence, strongest signal first:
 * 1. `merged: true` — a merged GitHub PR is `{ state: "closed", merged: true }`;
 *    without this the chip would render red instead of purple.
 * 2. Explicit non-"open" SuperPlane `state` (draft/closed/merged).
 * 3. `draft: true` — GitHub draft PRs stay `state: "open"`.
 * 4. Explicit "open" `state`.
 *
 * Returns undefined for missing / unrecognized values so the chip
 * falls back to the default "open" look instead of misrepresenting
 * the PR.
 */
export function extractPrArtifactState(data: ArtifactData): PrArtifactState | undefined {
  if (extractArtifactBoolean(data, "merged") === true) {
    return "merged";
  }

  // A flag-only update can leave `state: merged` in the map while writing
  // `merged: false`. The flag is the newer signal — do not keep purple.
  let explicit = readPrArtifactStateField(data);
  if (explicit === "merged" && extractArtifactBoolean(data, "merged") === false) {
    explicit = undefined;
  }
  if (explicit === "draft" && extractArtifactBoolean(data, "draft") === false) {
    explicit = undefined;
  }

  if (explicit && explicit !== "open") {
    return explicit;
  }

  if (extractArtifactBoolean(data, "draft") === true) {
    return "draft";
  }

  return explicit;
}

export function buildLatestArtifactDataById(
  artifacts: Array<{ id?: string; data?: Record<string, unknown> }> | undefined,
): Map<string, Record<string, unknown>> {
  const byId = new Map<string, Record<string, unknown>>();
  for (const artifact of artifacts ?? []) {
    if (artifact.id && artifact.data) {
      byId.set(artifact.id, artifact.data);
    }
  }
  return byId;
}

/** Overlay current artifact data on an attach-time snapshot. Falls back to the snapshot when no match exists. */
export function overlayLiveArtifactData<T extends { id?: string; data?: Record<string, unknown> }>(
  artifact: T,
  latestById: Map<string, Record<string, unknown>>,
): T {
  const liveData = artifact.id ? latestById.get(artifact.id) : undefined;
  return liveData ? { ...artifact, data: liveData } : artifact;
}

// A repository URL can carry credentials, a query, or a fragment
// (`https://oauth2:token@host/acme/repo?tab=readme`). Drop those before the
// branch path lands in an href, the same way the backend does.
function treeUrlFromRepositoryUrl(repository: string, branchPath: string): string | undefined {
  try {
    const parsed = new URL(repository);
    parsed.username = "";
    parsed.password = "";
    parsed.search = "";
    parsed.hash = "";
    parsed.pathname = `${parsed.pathname.replace(/\/+$/, "")}/tree/${branchPath}`;
    return parsed.toString();
  } catch {
    return undefined;
  }
}

function readPrArtifactStateField(data: ArtifactData): PrArtifactState | undefined {
  const raw = extractArtifactField(data, "state");
  if (!raw) {
    return undefined;
  }
  const normalized = raw.toLowerCase();
  return PR_ARTIFACT_STATES.find((state) => state === normalized);
}

// GitHub payloads carry real booleans; templated flow inputs almost
// always arrive as the strings "true" / "false". Tolerate both so a
// PR chip can pick up the state without an if-node in the flow.
function extractArtifactBoolean(data: ArtifactData, key: string): boolean | undefined {
  if (!data) {
    return undefined;
  }
  const value = data[key];
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "string") {
    const trimmed = value.trim().toLowerCase();
    if (trimmed === "true") return true;
    if (trimmed === "false") return false;
  }
  return undefined;
}

/**
 * Narrows the API's `data?: unknown` into the string-keyed shape every
 * artifact consumer expects, without lying about non-object payloads.
 */
export function toArtifactDataRecord(data: unknown): Record<string, unknown> | undefined {
  return data && typeof data === "object" ? (data as Record<string, unknown>) : undefined;
}

function readPositiveBytes(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return value;
  }
  if (typeof value === "string" && value.trim() !== "" && !/[A-Za-z]/.test(value)) {
    const parsed = Number(value);
    if (Number.isFinite(parsed) && parsed > 0) {
      return parsed;
    }
  }
  return undefined;
}

function formatSizeNumber(value: number): string {
  const rounded = value >= 10 ? Math.round(value) : Math.round(value * 10) / 10;
  return String(rounded);
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
