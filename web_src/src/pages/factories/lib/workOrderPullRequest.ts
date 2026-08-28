import type { CanvasesCanvasRunRef, FactoriesFactoryPullRequest, FactoriesFactoryPullRequestState } from "@/api-client";

export type FactoryPullRequestState = "open" | "draft" | "closed" | "merged";

export function groupPullRequestsByWorkOrderId(
  pullRequests: FactoriesFactoryPullRequest[],
): Map<string, FactoriesFactoryPullRequest[]> {
  const grouped = new Map<string, FactoriesFactoryPullRequest[]>();
  for (const pullRequest of pullRequests) {
    const workOrderId = pullRequest.workOrderId?.trim();
    if (!workOrderId) {
      continue;
    }
    const existing = grouped.get(workOrderId);
    if (existing) {
      existing.push(pullRequest);
      continue;
    }
    grouped.set(workOrderId, [pullRequest]);
  }
  return grouped;
}

export function isPullRequestArtifactType(type: string | undefined): boolean {
  return (type ?? "").replace(/^TYPE_/i, "").toLowerCase() === "pr";
}

export function withoutPullRequestArtifacts<T extends { type?: string }>(artifacts: T[]): T[] {
  return artifacts.filter((artifact) => !isPullRequestArtifactType(artifact.type));
}

export function pullRequestState(
  state: FactoriesFactoryPullRequestState | string | undefined,
): FactoryPullRequestState {
  switch (state) {
    case "STATE_DRAFT":
    case "draft":
      return "draft";
    case "STATE_CLOSED":
    case "closed":
      return "closed";
    case "STATE_MERGED":
    case "merged":
      return "merged";
    default:
      return "open";
  }
}

export function pullRequestLabel(pullRequest: FactoriesFactoryPullRequest): string {
  const number = String(pullRequest.number ?? "")
    .replace(/^#/, "")
    .trim();
  if (number) {
    return `#${number}`;
  }
  const title = pullRequest.title?.trim();
  if (title) {
    return title;
  }
  return "Pull request";
}

export function isActiveCanvasRun(run: CanvasesCanvasRunRef | undefined): boolean {
  if (!run?.id) {
    return false;
  }
  return run.state === "STATE_PENDING" || run.state === "STATE_STARTED" || run.state === "STATE_CANCELLING";
}

export function statusForCanvasRun(run: CanvasesCanvasRunRef | undefined): "passed" | "running" | "pending" | "failed" {
  if (run?.state === "STATE_STARTED" || run?.state === "STATE_CANCELLING") {
    return "running";
  }
  if (run?.state === "STATE_FINISHED") {
    if (run.result === "RESULT_PASSED") {
      return "passed";
    }
    if (run.result === "RESULT_FAILED" || run.result === "RESULT_CANCELLED") {
      return "failed";
    }
  }
  return "pending";
}

export function prFeedbackRunTitle(pullRequest: FactoriesFactoryPullRequest): string {
  const number = String(pullRequest.number ?? "")
    .replace(/^#/, "")
    .trim();
  if (number) {
    return `Address feedback on PR #${number}`;
  }
  return "Address PR feedback";
}

export function indexPullRequestsById(
  pullRequests: FactoriesFactoryPullRequest[] | undefined,
): Map<string, FactoriesFactoryPullRequest> {
  const indexed = new Map<string, FactoriesFactoryPullRequest>();
  for (const pullRequest of pullRequests ?? []) {
    if (pullRequest.id) {
      indexed.set(pullRequest.id, pullRequest);
    }
  }
  return indexed;
}

export function overlayLivePullRequest(
  snapshot: FactoriesFactoryPullRequest,
  liveById: Map<string, FactoriesFactoryPullRequest>,
): FactoriesFactoryPullRequest {
  if (!snapshot.id) {
    return snapshot;
  }
  const live = liveById.get(snapshot.id);
  return live ? { ...snapshot, ...live } : snapshot;
}

export function pullRequestFromEventPayload(payload: {
  id?: string;
  provider?: string;
  repository?: string;
  number?: number | string;
  url?: string;
  title?: string;
  state?: string;
}): FactoriesFactoryPullRequest {
  return {
    id: payload.id,
    provider: payload.provider as FactoriesFactoryPullRequest["provider"],
    repository: payload.repository,
    number: payload.number == null ? undefined : String(payload.number),
    url: payload.url,
    title: payload.title,
    state: protoPullRequestState(payload.state),
  };
}

function protoPullRequestState(state: string | undefined): FactoriesFactoryPullRequestState | undefined {
  if (!state?.trim()) {
    return undefined;
  }
  switch (pullRequestState(state)) {
    case "draft":
      return "STATE_DRAFT";
    case "closed":
      return "STATE_CLOSED";
    case "merged":
      return "STATE_MERGED";
    default:
      return "STATE_OPEN";
  }
}
