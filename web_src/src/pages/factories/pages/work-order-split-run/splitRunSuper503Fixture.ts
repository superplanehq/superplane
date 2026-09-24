import type {
  FactoriesFactoryPullRequest,
  FactoriesFactoryPullRequestRevision,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderCheck,
} from "@/api-client";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";

import { presentWorkOrderChecks } from "../../lib/workOrderChecks";
import { doneFooterForStatus } from "./splitRunFooter";
import type { SplitRunFixture, SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";
import {
  SUPER503_MODEL,
  super503AnalysisStream,
  super503BacklogStream,
  super503ChecksStream,
  super503ClosureStream,
  super503ImplementStream,
  super503ReviewStream,
  super503StorybookStream,
  super503Time,
  type Super503ReviewLogKey,
} from "./splitRunSuper503Streams";

/**
 * Storybook fixture from SUPER-503, "Dictation papercuts", on the
 * Superplane Steam Machine line. Analysis wrote a plan and a confidence
 * score, Implement opened PR #7771, five Greptile rounds followed, and
 * PR Closure closed the task when the pull request merged.
 */

const REPOSITORY = "superplanehq/superplane";
const BRANCH = "fix/put-dictation-in-input";
const PR_NUMBER = "7771";
const PR_URL = `https://github.com/${REPOSITORY}/pull/${PR_NUMBER}`;
const STORYBOOK_URL = "https://storybook-pr-7771.superplane.workers.dev";
const TITLE = "Dictation papercuts";

const WORK_ORDER_ID = "53d9b9fa-133c-4db7-9319-57a0a8017058";
const IMPLEMENT_APP_ID = "b60a7177-9bb7-47eb-a499-6de946f14190";
const IMPLEMENT_RUN_ID = "92957aab-b683-43ef-9cf6-747ab6fcaff1";
const ANALYSIS_APP_ID = "0fc7a286-51b7-449c-812c-62093506c5fd";
const ANALYSIS_RUN_ID = "f6c3162b-9311-4ecb-81ce-78a0906cbb6e";
const PR_CHECKS_APP_ID = "bec62734-3b6a-4ced-b9b3-1771dcb8ab1f";
const STORYBOOK_APP_ID = "a2eb05c0-f706-40d8-a237-c0430fccdf52";
const PR_REVIEW_APP_ID = "b5623f9e-276e-467f-8e2e-863632623a1a";
const CLOSURE_APP_ID = "c803ac0a-1e87-4901-988b-9b9f141315dc";
const CLOSURE_RUN_ID = "e0b55e20-133a-4142-9f13-e991335c066a";

const OWNER_NAME = "Aleksandar Mitrovic";

const OWNER: OrgUserDisplay = {
  id: "b69e6f19-7830-4000-ae4e-1206572032c2",
  name: OWNER_NAME,
  initials: getUserInitials(OWNER_NAME),
};

const DESCRIPTION_TEXT = `Currently on the auto dictation thing on create task but also in the chat I think the button is moving along with the transcript it's very weird to check the screenshot. The button should stay on the left. It is also moving some reason the screenshot icons along with the button, it's weird.

I also don't like that the transcript is not that visible if there is more than one line it gets truncated at some point so I don't see the full transcript.`;

const PLAN_TEXT = `# Put speech in the input

Live speech no longer sits beside the mic. In-progress and finished words appear in the chat box or create-task field. The attach, mic, and screenshot controls stay still.

## Problem
A live transcript sits in the toolbar in front of the stop control. As it grows, the mic and screenshots slide. The same words later appear in the field. The extra line is easy to miss and easy to clip.

## Proposed outcome
There is no transcript in the toolbar. As the user speaks, the words appear in the focused create-task field or in the chat box. Finished phrases stay there. Attach, mic, and screenshot controls stay on the left.`;

const DESCRIPTION_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-503-description",
  type: "TYPE_MARKDOWN",
  data: { name: "description.md", title: "description.md", body: DESCRIPTION_TEXT },
};

const PLAN_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-503-plan",
  type: "TYPE_MARKDOWN",
  data: { name: "plan.md", title: "plan.md", body: PLAN_TEXT },
};

const BRANCH_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-503-branch",
  type: "TYPE_BRANCH",
  data: { name: BRANCH, repository: REPOSITORY, url: `https://github.com/${REPOSITORY}/tree/${BRANCH}` },
};

const STORYBOOK_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-503-storybook",
  type: "TYPE_LINK",
  data: { title: "Storybook deployment", url: STORYBOOK_URL },
};

const EVIDENCE_IMAGE: FactoriesWorkOrderArtifact = {
  id: "art-503-poster",
  type: "TYPE_FILE",
  data: { title: "Live speech in the create-task field poster", filename: "dictation-in-field.png" },
};

const EVIDENCE_VIDEO: FactoriesWorkOrderArtifact = {
  id: "art-503-video",
  type: "TYPE_FILE",
  data: { title: "Live speech in the create-task field", filename: "dictation-in-field.webm" },
};

const PULL_REQUEST: FactoriesFactoryPullRequest = {
  id: "1c6f1a82-b1d4-4b54-9ca3-4fcc9d8cee43",
  workOrderId: WORK_ORDER_ID,
  provider: "PROVIDER_GITHUB",
  repository: REPOSITORY,
  number: PR_NUMBER,
  url: PR_URL,
  title: "fix: Put live dictation in the input field",
  state: "STATE_MERGED",
};

function revision(sha: string, at: string): FactoriesFactoryPullRequestRevision {
  return { id: `rev-503-${sha}`, sha, createdAt: super503Time(at) };
}

const SHA = {
  initial: revision("050aaec", "11:56:57"),
  focus: revision("c9985a9", "12:11:01"),
  duplicate: revision("0310281", "12:24:13"),
  retained: revision("bc420ea", "12:36:27"),
  partial: revision("70dd134", "12:48:58"),
  render: revision("99951af", "13:00:34"),
};

const CONFIDENCE_CHECK: FactoriesWorkOrderCheck = {
  id: "check-503-confidence",
  key: "confidence",
  name: "Confidence score",
  score: 4,
  maxScore: 5,
  level: "LEVEL_POSITIVE",
  summary: "Confidence score 4/5.",
  automation: { appId: ANALYSIS_APP_ID, appName: "Refine Task" },
  runId: ANALYSIS_RUN_ID,
  updatedAt: super503Time("11:33:11"),
};

const ANALYSIS_CHECKS = presentWorkOrderChecks([CONFIDENCE_CHECK]);

function phase(input: {
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
  revision?: FactoriesFactoryPullRequestRevision;
  checks?: SplitRunPhase["checks"];
}): SplitRunPhase {
  const startedAt = super503Time(input.startedAt);
  return {
    id: input.id,
    name: input.name,
    description: input.description,
    status: "passed",
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
    pullRequestActivity: input.revision
      ? { pullRequest: PULL_REQUEST, revision: input.revision, startedAt }
      : undefined,
  };
}

function checksPhase(input: {
  runId: string;
  sha: string;
  startedAt: string;
  duration: string;
  waitDuration: string;
  revision: FactoriesFactoryPullRequestRevision;
}): SplitRunPhase {
  return phase({
    id: `pr-feedback-${input.runId}`,
    name: `Checks passed on ${input.sha}`,
    description: "`ci/semaphoreci/pr: CI` — The build passed on Semaphore 2.0.",
    componentName: "PR Checks",
    duration: input.duration,
    startedAt: input.startedAt,
    stream: super503ChecksStream({ prefix: `checks-${input.sha}`, at: input.startedAt, waitDuration: input.waitDuration }),
    appId: PR_CHECKS_APP_ID,
    runId: input.runId,
    revision: input.revision,
  });
}

function storybookPhase(input: {
  runId: string;
  startedAt: string;
  duration: string;
  deployDuration: string;
  revision: FactoriesFactoryPullRequestRevision;
}): SplitRunPhase {
  return phase({
    id: `pr-feedback-${input.runId}`,
    name: "Storybook deployment ready",
    description: `Preview at ${STORYBOOK_URL}`,
    componentName: "Deploy Storybook",
    duration: input.duration,
    startedAt: input.startedAt,
    stream: super503StorybookStream({
      prefix: `storybook-${input.revision.sha}`,
      at: input.startedAt,
      duration: input.deployDuration,
      link: STORYBOOK_ARTIFACT,
    }),
    appId: STORYBOOK_APP_ID,
    runId: input.runId,
    costCents: "1",
    revision: input.revision,
  });
}

function reviewPhase(input: {
  runId: string;
  log: Super503ReviewLogKey;
  description: string;
  startedAt: string;
  duration: string;
  agentDuration: string;
  costCents: string;
  totalTokens: string;
  revision: FactoriesFactoryPullRequestRevision;
}): SplitRunPhase {
  return phase({
    id: `pr-feedback-${input.runId}`,
    name: "Greptile left a review",
    description: input.description,
    componentName: "Address PR Review",
    duration: input.duration,
    startedAt: input.startedAt,
    stream: super503ReviewStream({
      prefix: input.log,
      at: input.startedAt,
      agentDuration: input.agentDuration,
      log: input.log,
    }),
    appId: PR_REVIEW_APP_ID,
    runId: input.runId,
    costCents: input.costCents,
    totalTokens: input.totalTokens,
    model: SUPER503_MODEL,
    revision: input.revision,
  });
}

const TASK_PHASES: SplitRunPhase[] = [
  phase({
    id: "backlog",
    name: "Backlog",
    componentName: "Created manually",
    duration: "1s",
    startedAt: "11:28:53",
    artifacts: [DESCRIPTION_ARTIFACT],
    stream: super503BacklogStream(DESCRIPTION_ARTIFACT),
  }),
  phase({
    id: "analysis",
    name: "Analysis",
    componentName: "Refine Task",
    duration: "4m 8s",
    startedAt: "11:29:06",
    artifacts: [PLAN_ARTIFACT],
    checks: ANALYSIS_CHECKS,
    stream: super503AnalysisStream(PLAN_ARTIFACT),
    appId: ANALYSIS_APP_ID,
    runId: ANALYSIS_RUN_ID,
    costCents: "64",
    totalTokens: "489691",
    model: SUPER503_MODEL,
  }),
  phase({
    id: "implement",
    name: "Implement",
    componentName: "Implement",
    duration: "19m 50s",
    startedAt: "11:37:08",
    artifacts: [BRANCH_ARTIFACT, EVIDENCE_IMAGE, EVIDENCE_VIDEO],
    stream: super503ImplementStream(BRANCH_ARTIFACT, PULL_REQUEST),
    appId: IMPLEMENT_APP_ID,
    runId: IMPLEMENT_RUN_ID,
    stepIndex: 0,
    costCents: "733",
    totalTokens: "11482898",
    model: SUPER503_MODEL,
  }),
  phase({
    id: "done",
    name: "Done",
    description: "PR #7771 was merged.",
    componentName: "PR Closure",
    duration: "2s",
    startedAt: "13:11:07",
    stream: super503ClosureStream(PULL_REQUEST),
    appId: CLOSURE_APP_ID,
    runId: CLOSURE_RUN_ID,
  }),
];

const PULL_REQUEST_PHASES: SplitRunPhase[] = [
  checksPhase({
    runId: "eb5b062e-ec2a-45e4-9fe2-979fbc72b081",
    sha: "050aaec",
    startedAt: "11:56:58",
    duration: "7m 16s",
    waitDuration: "7m 16s",
    revision: SHA.initial,
  }),
  storybookPhase({
    runId: "26ca3895-2502-42a7-a567-376d0e6e31fa",
    startedAt: "11:56:58",
    duration: "1m 12s",
    deployDuration: "1m 12s",
    revision: SHA.initial,
  }),
  reviewPhase({
    runId: "56e9455a-0b43-40c7-b917-cbc46b331319",
    log: "review1",
    description:
      "P1 on `useWorkOrderFieldDictation.ts`: **Field switching overwrites existing text**. Focusing the title while dictating into the description updates only `lastFieldRef`.",
    startedAt: "12:01:35",
    duration: "9m 32s",
    agentDuration: "9m 30s",
    costCents: "165",
    totalTokens: "1781630",
    revision: SHA.initial,
  }),
  checksPhase({
    runId: "ee0dae57-6a4b-4b30-843a-47a4c6bf9994",
    sha: "c9985a9",
    startedAt: "12:11:01",
    duration: "7m 21s",
    waitDuration: "7m 21s",
    revision: SHA.focus,
  }),
  storybookPhase({
    runId: "432223ff-1d8e-433a-a03e-9d583a567ac2",
    startedAt: "12:11:01",
    duration: "1m 21s",
    deployDuration: "1m 21s",
    revision: SHA.focus,
  }),
  reviewPhase({
    runId: "4c68e5c0-07aa-4af0-93e5-ce4a80fc74c0",
    log: "review2",
    description:
      "P1 on `useSpokenPhraseDictation.ts`: **Returning focus duplicates spoken words**. Switching away and back commits provisional words too early.",
    startedAt: "12:15:07",
    duration: "9m 11s",
    agentDuration: "9m 9s",
    costCents: "162",
    totalTokens: "2039100",
    revision: SHA.focus,
  }),
  checksPhase({
    runId: "8a6af718-f43b-4cfd-b2dc-8dd724c44aa0",
    sha: "0310281",
    startedAt: "12:24:13",
    duration: "6m 28s",
    waitDuration: "6m 28s",
    revision: SHA.duplicate,
  }),
  storybookPhase({
    runId: "4704d4a8-9c43-40e0-9b3b-f3bbbac272ea",
    startedAt: "12:24:13",
    duration: "1m 26s",
    deployDuration: "1m 26s",
    revision: SHA.duplicate,
  }),
  reviewPhase({
    runId: "8db5e69f-4205-43d4-9320-377a1c14efa3",
    log: "review3",
    description:
      "P1 on `useSpokenPhraseDictation.ts`: **Returning focus discards retained words**. Interim text finalized in the title disappears from the description.",
    startedAt: "12:28:00",
    duration: "8m 33s",
    agentDuration: "8m 31s",
    costCents: "136",
    totalTokens: "1807569",
    revision: SHA.duplicate,
  }),
  checksPhase({
    runId: "f61c7742-4833-4c34-8f3d-e332bdbea750",
    sha: "bc420ea",
    startedAt: "12:36:27",
    duration: "8m 37s",
    waitDuration: "8m 37s",
    revision: SHA.retained,
  }),
  storybookPhase({
    runId: "05cc479d-d095-4434-8c08-80b1a2bf958c",
    startedAt: "12:36:27",
    duration: "1m 18s",
    deployDuration: "1m 18s",
    revision: SHA.retained,
  }),
  reviewPhase({
    runId: "84b46f90-3a0f-4c43-a9fe-9a9833068c5b",
    log: "review4",
    description:
      "P1 on `useSpokenPhraseDictation.ts`: **Partial finalization duplicates pending words**. A final result does not finish the whole saved interim phrase.",
    startedAt: "12:41:01",
    duration: "8m 6s",
    agentDuration: "8m 4s",
    costCents: "128",
    totalTokens: "1565814",
    revision: SHA.retained,
  }),
  checksPhase({
    runId: "a49940b1-6c51-4076-91c7-6660a008ecd2",
    sha: "70dd134",
    startedAt: "12:48:58",
    duration: "8m 14s",
    waitDuration: "8m 14s",
    revision: SHA.partial,
  }),
  storybookPhase({
    runId: "dfe87e05-8243-40b4-be71-a880e78bdd0c",
    startedAt: "12:48:58",
    duration: "1m 21s",
    deployDuration: "1m 21s",
    revision: SHA.partial,
  }),
  reviewPhase({
    runId: "b95f0016-1ec7-44f3-ad1f-a19bcc3e8031",
    log: "review5",
    description:
      "P2 on `useSpokenPhraseDictation.ts`: **Parent synchronization runs during render**. `syncFromField()` copies parent values into mutable snapshots during render.",
    startedAt: "12:53:59",
    duration: "6m 40s",
    agentDuration: "6m 38s",
    costCents: "113",
    totalTokens: "1534156",
    revision: SHA.partial,
  }),
  checksPhase({
    runId: "cb916cf0-a8d0-4bc6-85d3-6b22bff29107",
    sha: "99951af",
    startedAt: "13:00:34",
    duration: "7m 2s",
    waitDuration: "7m 2s",
    revision: SHA.render,
  }),
  storybookPhase({
    runId: "8185fc64-a7b2-4f8b-9ec3-c04932799966",
    startedAt: "13:00:34",
    duration: "3m 56s",
    deployDuration: "3m 56s",
    revision: SHA.render,
  }),
];

export const SPLIT_RUN_SUPER503: SplitRunFixture = {
  title: TITLE,
  descriptionText: DESCRIPTION_TEXT,
  owner: OWNER,
  assigneeIds: [OWNER.id],
  elapsed: "1h 16m",
  startedLabel: "Started yesterday",
  costUsd: "$15.07",
  tokensLabel: "20.7M tokens",
  usageByModel: [{ provider: "openrouter", model: "x-ai/grok-4.6", totalTokens: "20700858", costCents: "1507" }],
  lineName: "Superplane Steam Machine",
  currentStepIndex: 0,
  lineStatus: "passed",
  currentPhaseId: "done",
  openPhaseId: "implement",
  phases: [...TASK_PHASES, ...PULL_REQUEST_PHASES],
  waitingNotes: [],
  checks: ANALYSIS_CHECKS,
  footer: {
    ...doneFooterForStatus("completed", { automationName: "PR Closure" }),
    run: { appId: CLOSURE_APP_ID, runId: CLOSURE_RUN_ID },
  },
  footerTone: "done",
};
