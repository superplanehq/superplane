import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import type { SplitRunPhase, SplitRunRevision, SplitRunStreamLine } from "./splitRunMocks";
import { super503Time } from "./splitRunSuper503Streams";

/**
 * Identifiers and the phase builder shared by the SUPER-503 fixtures.
 * Every id here is the real one from production; the pull request phases
 * and the task phases both need them.
 */

export const SUPER503_REPOSITORY = "superplanehq/superplane";
export const SUPER503_PR_NUMBER = "7771";
export const SUPER503_PR_URL = `https://github.com/${SUPER503_REPOSITORY}/pull/${SUPER503_PR_NUMBER}`;
export const SUPER503_STORYBOOK_URL = "https://storybook-pr-7771.superplane.workers.dev";
export const SUPER503_WORK_ORDER_ID = "53d9b9fa-133c-4db7-9319-57a0a8017058";

/** Factory automations of "SuperPlane Prod" by their real canvas ids. */
export const SUPER503_APPS = {
  backlog: "0fc7a286-51b7-449c-812c-62093506c5fd",
  implement: "b60a7177-9bb7-47eb-a499-6de946f14190",
  fixChecks: "bec62734-3b6a-4ced-b9b3-1771dcb8ab1f",
  storybook: "a2eb05c0-f706-40d8-a237-c0430fccdf52",
  addressFeedback: "b5623f9e-276e-467f-8e2e-863632623a1a",
  closure: "c803ac0a-1e87-4901-988b-9b9f141315dc",
} as const;

/** Names of those automations as the factory lists them. */
export const SUPER503_AUTOMATION_NAMES = {
  backlog: "Backlog",
  implement: "Implement",
  fixChecks: "Fix pull request checks",
  storybook: "Deploys Storybook",
  addressFeedback: "Address PR feedback",
  closure: "PR Closure",
} as const;

export const SUPER503_IMPLEMENT_RUN = {
  appId: SUPER503_APPS.implement,
  runId: "92957aab-b683-43ef-9cf6-747ab6fcaff1",
};

export const SUPER503_CLOSURE_RUN = {
  appId: SUPER503_APPS.closure,
  runId: "e0b55e20-133a-4142-9f13-e991335c066a",
};

export const SUPER503_PULL_REQUEST: FactoriesFactoryPullRequest = {
  id: "1c6f1a82-b1d4-4b54-9ca3-4fcc9d8cee43",
  workOrderId: SUPER503_WORK_ORDER_ID,
  provider: "PROVIDER_GITHUB",
  repository: SUPER503_REPOSITORY,
  number: SUPER503_PR_NUMBER,
  url: SUPER503_PR_URL,
  title: "fix: Put live dictation in the input field",
  state: "STATE_MERGED",
};

export const SUPER503_STORYBOOK_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-503-storybook",
  type: "TYPE_LINK",
  data: { title: "Storybook deployment", url: SUPER503_STORYBOOK_URL },
};

export function super503Phase(input: {
  id: string;
  name: string;
  componentName: string;
  duration: string;
  startedAt: string;
  stream: SplitRunStreamLine[];
  description?: string;
  artifacts?: FactoriesWorkOrderArtifact[];
  appId?: string;
  runId?: string;
  stepIndex?: number;
  costCents?: string;
  totalTokens?: string;
  model?: string;
  /** Commit that started this pull request activity. */
  revision?: SplitRunRevision;
  /** Pull request activity that no commit started, such as the merge. */
  onPullRequest?: boolean;
  checks?: SplitRunPhase["checks"];
  /** Defaults to `passed`. */
  status?: SplitRunPhase["status"];
}): SplitRunPhase {
  const startedAt = super503Time(input.startedAt);
  const pullRequestActivity =
    input.revision || input.onPullRequest
      ? { pullRequest: SUPER503_PULL_REQUEST, revision: input.revision, startedAt }
      : undefined;
  return {
    id: input.id,
    name: input.name,
    description: input.description,
    status: input.status ?? "passed",
    duration: input.duration,
    startedAt,
    componentName: input.componentName,
    artifacts: input.artifacts ?? [],
    checks: input.checks,
    stream: input.stream,
    canvasSteps: [],
    canvasKey: null,
    appId: input.appId,
    runId: input.runId,
    stepIndex: input.stepIndex,
    costCents: input.costCents,
    totalTokens: input.totalTokens,
    model: input.model,
    pullRequestActivity,
  };
}
