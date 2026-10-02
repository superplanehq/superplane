import type { CanvasesCanvasRun } from "@/api-client";
import { mergeCanvasRunUpdate } from "@/hooks/canvasInfiniteCache";

import { isMergeConfidenceCanvas } from "./columnAutomations";

/** One Merge confidence canvas run that belongs to a task pull request. */
export type MergeConfidenceLogRun = {
  canvasId: string;
  canvasName: string;
  run: CanvasesCanvasRun;
};

export type MergeConfidenceCanvas = {
  id: string;
  name: string;
};

type PullRequestRef = {
  number?: string | number;
  repository?: string;
  url?: string;
};

type PullRequestIdentity = {
  number?: string;
  repository?: string;
  url?: string;
};

const GITHUB_PULL_URL = /^https?:\/\/[^/]+\/([^/]+)\/([^/]+)\/pull\/(\d+)/i;

/** Verify canvases that score merge confidence, in factory app order. */
export function mergeConfidenceCanvases(
  apps: Array<{ id?: string; name?: string; columnKey?: string }>,
): MergeConfidenceCanvas[] {
  return apps.flatMap((app) => {
    const id = app.id?.trim();
    if (!id || !isMergeConfidenceCanvas(app)) {
      return [];
    }
    return [{ id, name: app.name?.trim() || "Merge confidence" }];
  });
}

/**
 * Runs of one canvas that score a task pull request. A match needs the
 * pull request number and repository, or the same pull request URL.
 */
export function mergeConfidenceRunsForPullRequests(
  canvas: MergeConfidenceCanvas,
  runs: CanvasesCanvasRun[],
  pullRequests: PullRequestRef[],
): MergeConfidenceLogRun[] {
  const tasks = pullRequests.map(pullRequestIdentity);
  return runs
    .flatMap((run) => {
      if (!run.id || !tasks.some((task) => identitiesMatch(pullRequestIdentityFromRootEvent(run), task))) {
        return [];
      }
      return [{ canvasId: canvas.id, canvasName: canvas.name, run }];
    })
    .sort((left, right) => Date.parse(left.run.createdAt ?? "") - Date.parse(right.run.createdAt ?? ""));
}

/** Pull request number, repository, and URL from the trigger envelope. */
export function pullRequestIdentityFromRootEvent(run: CanvasesCanvasRun): PullRequestIdentity {
  const payload = eventPayload(run);
  const pullRequest = asRecord(payload?.pull_request);
  const url = pullRequestUrl(pullRequest);
  const fromUrl = identityFromPullUrl(url);
  return {
    number: eventPullRequestNumber(payload, pullRequest) ?? fromUrl.number,
    repository: eventRepository(payload, pullRequest) ?? fromUrl.repository,
    url,
  };
}

function eventPayload(run: CanvasesCanvasRun): Record<string, unknown> | undefined {
  const envelope = asRecord(run.rootEvent?.data);
  return asRecord(envelope?.data) ?? envelope;
}

function eventPullRequestNumber(
  payload: Record<string, unknown> | undefined,
  pullRequest: Record<string, unknown> | undefined,
): string | undefined {
  return normalizePullRequestNumber(pullRequest?.number ?? payload?.number);
}

function eventRepository(
  payload: Record<string, unknown> | undefined,
  pullRequest: Record<string, unknown> | undefined,
): string | undefined {
  const repository = asRecord(payload?.repository) ?? baseRepository(pullRequest);
  return normalizeRepository(repositoryName(repository));
}

function baseRepository(pullRequest: Record<string, unknown> | undefined): Record<string, unknown> | undefined {
  return asRecord(asRecord(pullRequest?.base)?.repo);
}

function pullRequestUrl(pullRequest: Record<string, unknown> | undefined): string | undefined {
  return normalizePullRequestUrl(stringValue(pullRequest?.html_url) ?? stringValue(pullRequest?.url));
}

/** Apply one live run payload without losing newer state already in the cache. */
export function upsertMergeConfidenceCanvasRun(
  current: CanvasesCanvasRun[] | undefined,
  run: CanvasesCanvasRun,
): CanvasesCanvasRun[] {
  if (!run.id) {
    return current ?? [];
  }
  if (!current) {
    return [run];
  }
  const index = current.findIndex((entry) => entry.id === run.id);
  if (index < 0) {
    return sortCanvasRuns([...current, run]);
  }
  const merged = mergeCanvasRunUpdate(current[index], run);
  if (merged === current[index]) {
    return current;
  }
  const next = [...current];
  next[index] = merged;
  return next;
}

/**
 * Apply a REST snapshot. Keep a live run that arrived during the fetch.
 * Drop a cached run that the snapshot no longer includes.
 */
export function mergeMergeConfidenceRunSnapshots(
  current: CanvasesCanvasRun[] | undefined,
  incoming: CanvasesCanvasRun[],
  idsAtFetchStart?: ReadonlySet<string>,
): CanvasesCanvasRun[] {
  const incomingIds = new Set(incoming.flatMap((run) => (run.id ? [run.id] : [])));
  let merged = incoming;
  for (const run of current ?? []) {
    if (!run.id) {
      continue;
    }
    const arrivedDuringFetch = idsAtFetchStart !== undefined && !idsAtFetchStart.has(run.id);
    if (!incomingIds.has(run.id) && !arrivedDuringFetch) {
      continue;
    }
    merged = upsertMergeConfidenceCanvasRun(merged, run);
  }
  return merged;
}

function identitiesMatch(event: PullRequestIdentity, task: PullRequestIdentity): boolean {
  if (event.url && task.url && event.url === task.url) {
    return true;
  }
  if (!event.number || !task.number || !event.repository || !task.repository) {
    return false;
  }
  return event.number === task.number && event.repository === task.repository;
}

function pullRequestIdentity(pullRequest: PullRequestRef): PullRequestIdentity {
  const url = normalizePullRequestUrl(pullRequest.url);
  const fromUrl = identityFromPullUrl(url);
  return {
    number: normalizePullRequestNumber(pullRequest.number) ?? fromUrl.number,
    repository: normalizeRepository(pullRequest.repository) ?? fromUrl.repository,
    url,
  };
}

function identityFromPullUrl(url: string | undefined): Pick<PullRequestIdentity, "number" | "repository"> {
  if (!url) {
    return {};
  }
  const match = url.match(GITHUB_PULL_URL);
  if (!match) {
    return {};
  }
  return {
    repository: normalizeRepository(`${match[1]}/${match[2]}`),
    number: match[3],
  };
}

function repositoryName(repository: Record<string, unknown> | undefined): string | undefined {
  const fullName = stringValue(repository?.full_name);
  if (fullName?.includes("/")) {
    return fullName;
  }
  const login = stringValue(asRecord(repository?.owner)?.login);
  const name = stringValue(repository?.name);
  if (login && name) {
    return `${login}/${name}`;
  }
  return fullName;
}

function normalizePullRequestNumber(value: unknown): string | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(Math.trunc(value));
  }
  if (typeof value !== "string") {
    return undefined;
  }
  const trimmed = value.trim().replace(/^#/, "");
  return /^\d+$/.test(trimmed) ? trimmed : undefined;
}

function normalizeRepository(value: string | undefined): string | undefined {
  const trimmed = value?.trim().toLowerCase();
  return trimmed || undefined;
}

function normalizePullRequestUrl(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  if (!trimmed) {
    return undefined;
  }
  const withoutHash = trimmed.split("#")[0]?.split("?")[0]?.replace(/\/+$/, "");
  return withoutHash?.toLowerCase() || undefined;
}

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

function sortCanvasRuns(runs: CanvasesCanvasRun[]): CanvasesCanvasRun[] {
  return [...runs].sort((left, right) => Date.parse(left.createdAt ?? "") - Date.parse(right.createdAt ?? ""));
}
