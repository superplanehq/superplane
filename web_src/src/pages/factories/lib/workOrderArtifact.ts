export type ArtifactData = Record<string, unknown> | undefined;

/** GitHub-style pull request lifecycle states a PR artifact's chip can render. */
export type PrArtifactState = "open" | "draft" | "closed" | "merged";

const PR_ARTIFACT_STATES: readonly PrArtifactState[] = ["open", "draft", "closed", "merged"];

// Keys we accept for the PR number in the free-form artifact data map.
// The Add Work Order Artifact component's example uses `number`; some
// authors reach for `prNumber`. Tolerate both so links render as `#1234`
// regardless of which convention was used.
const PR_NUMBER_KEYS = ["number", "prNumber"] as const;

// Shape common to every place we look at sibling artifacts (sidebar,
// timeline events, dispatch steps). We only need the type + free-form
// data; consumers should not have to build a full presentation object.
export interface WorkOrderArtifactLike {
  id?: string;
  type?: string;
  data?: Record<string, unknown>;
}

/** Chip/list presentation: same as a sibling artifact, with a required type. */
export interface WorkOrderArtifactPresentation extends WorkOrderArtifactLike {
  type: string;
}

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

export function extractArtifactTitle(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "title");
}

export function extractArtifactName(data: ArtifactData): string | undefined {
  return extractArtifactField(data, "name");
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

/**
 * Returns a safe-looking external URL for a branch artifact, or undefined
 * when we can't produce one.
 *
 * Branches usually get attached before the PR exists, so `data.url` is
 * frequently missing. When a sibling PR artifact carries a GitHub-style
 * pull request URL (`https://{host}/{owner}/{repo}/pull/{n}`), we build
 * `https://{host}/{owner}/{repo}/tree/{branch}` from it and make the
 * chip clickable. Non-GitHub-style URLs are ignored so we don't ship a
 * broken link for GitLab / Bitbucket, which use different path prefixes.
 *
 * The caller still passes the result through `safeExternalUrl` before
 * putting it into an `href`.
 */
export function resolveBranchArtifactUrl(
  branchData: ArtifactData,
  relatedArtifacts?: readonly WorkOrderArtifactLike[],
): string | undefined {
  const direct = extractArtifactUrl(branchData);
  if (direct) {
    return direct;
  }

  const branchName = extractArtifactName(branchData);
  if (!branchName) {
    return undefined;
  }

  const candidates: Array<{ treeUrl: string; headRef?: string }> = [];
  for (const related of relatedArtifacts ?? []) {
    if (normalizeArtifactKind(related.type ?? "") !== "pr") {
      continue;
    }
    const siblingUrl = extractArtifactUrl(related.data);
    if (!siblingUrl) {
      continue;
    }

    const treeUrl = buildTreeUrlFromPullRequestUrl(siblingUrl, branchName);
    if (treeUrl) {
      candidates.push({ treeUrl, headRef: extractPrHeadRef(related.data) });
    }
  }

  const matching = candidates.filter((candidate) => candidate.headRef === branchName);
  if (matching.length > 0) {
    return matching[0].treeUrl;
  }

  // One GitHub PR on the work order: safe to assume it is the same repo.
  // Two or more with no head-ref match: do not guess the repository.
  if (candidates.length === 1) {
    return candidates[0].treeUrl;
  }

  return undefined;
}

/**
 * Strips the `TYPE_` proto prefix and lower-cases the artifact type so
 * every consumer branches on the same set of strings ("pr", "branch",
 * "markdown", ...). Exported because sibling-lookup logic outside this
 * file also needs it.
 */
export function normalizeArtifactKind(type: string): string {
  return type.replace(/^TYPE_/i, "").toLowerCase();
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

function readPrArtifactStateField(data: ArtifactData): PrArtifactState | undefined {
  const raw = extractArtifactField(data, "state");
  if (!raw) {
    return undefined;
  }
  const normalized = raw.toLowerCase();
  return PR_ARTIFACT_STATES.find((state) => state === normalized);
}

function extractPrHeadRef(data: ArtifactData): string | undefined {
  const fromField =
    extractArtifactField(data, "head_ref") ??
    extractArtifactField(data, "headRef") ??
    extractArtifactField(data, "branch");
  if (fromField) {
    return fromField;
  }
  if (!data) {
    return undefined;
  }

  const head = data.head;
  if (head && typeof head === "object" && !Array.isArray(head)) {
    const ref = (head as Record<string, unknown>).ref;
    if (typeof ref === "string" && ref.trim() !== "") {
      return ref.trim();
    }
  }
  return undefined;
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

function buildTreeUrlFromPullRequestUrl(pullUrl: string, branchName: string): string | undefined {
  let parsed: URL;
  try {
    parsed = new URL(pullUrl);
  } catch {
    return undefined;
  }

  const segments = parsed.pathname.split("/").filter(Boolean);
  const pullIndex = segments.findIndex((segment) => segment.toLowerCase() === "pull");
  if (pullIndex < 2) {
    return undefined;
  }

  const owner = segments[pullIndex - 2];
  const repo = segments[pullIndex - 1];
  if (!owner || !repo) {
    return undefined;
  }

  const encodedBranch = branchName
    .split("/")
    .map((segment) => encodeURIComponent(segment))
    .join("/");

  return `${parsed.origin}/${owner}/${repo}/tree/${encodedBranch}`;
}

/**
 * Narrows the API's `data?: unknown` into the string-keyed shape every
 * artifact consumer expects, without lying about non-object payloads.
 */
export function toArtifactDataRecord(data: unknown): Record<string, unknown> | undefined {
  return data && typeof data === "object" ? (data as Record<string, unknown>) : undefined;
}

// The API and timeline both carry artifacts with `data` typed as
// `unknown`, but the resolver only needs `{ type, data }`. Do the
// narrowing once at the call site so consumers can pass the array
// straight through.
export function toWorkOrderArtifactLikes(
  artifacts: ReadonlyArray<{ type?: string | null; data?: unknown }> | undefined,
): WorkOrderArtifactLike[] {
  return (artifacts ?? []).map((artifact) => ({
    type: artifact.type ?? undefined,
    data: toArtifactDataRecord(artifact.data),
  }));
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
