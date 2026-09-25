import type { SplitRunPhase, SplitRunRevision } from "./splitRunMocks";
import {
  SUPER503_APPS,
  SUPER503_AUTOMATION_NAMES,
  SUPER503_CLOSURE_RUN,
  SUPER503_STORYBOOK_ARTIFACT,
  SUPER503_STORYBOOK_URL,
  super503Phase,
} from "./splitRunSuper503Shared";
import {
  super503ChecksStream,
  super503ClosureStream,
  super503ReviewStream,
  super503StorybookStream,
  super503Time,
  type Super503ReviewLogKey,
} from "./splitRunSuper503Streams";
import review1 from "./super503/review-1.md?raw";
import review2 from "./super503/review-2.md?raw";
import review3 from "./super503/review-3.md?raw";
import review4 from "./super503/review-4.md?raw";
import review5 from "./super503/review-5.md?raw";

/**
 * The eighteen pull request activity rows of SUPER-503 as production
 * stores them: titles, descriptions, revisions, spend, and durations from
 * `pullRequests[0].activities`. Three automations took part: Fix pull
 * request checks and Deploys Storybook ran on each of six commits, and
 * Address PR feedback answered five Greptile reviews. PR Closure closed
 * the task when the pull request merged.
 */

const GREPTILE_REVIEW_TITLE = "[@greptile-apps[bot]](https://github.com/apps/greptile-apps) left a [review]";

function revision(sha: string, at: string, message: string, pushedBy: string): SplitRunRevision {
  return { id: `rev-503-${sha.slice(0, 7)}`, sha, createdAt: super503Time(at), message, pushedBy };
}

/** Commits on PR #7771. Implement pushed the first; each review fix pushed the next. */
const SHA = {
  initial: revision(
    "050aaecb6f77171fc47d237b2ee32baa4e23ab25",
    "11:56:57",
    "fix: Put live dictation in the input field",
    "Implement",
  ),
  focus: revision(
    "c9985a9e1ee4eb8388313bfcbf14c8c34f69cfcd",
    "12:11:01",
    "fix: Keep dictation snapshots per field",
    SUPER503_AUTOMATION_NAMES.addressFeedback,
  ),
  duplicate: revision(
    "0310281145097634adf72871039cb0a865ad58d9",
    "12:24:13",
    "fix: Restore live words after field return",
    SUPER503_AUTOMATION_NAMES.addressFeedback,
  ),
  retained: revision(
    "bc420ea0b6870bd6fff37b91b407d8a228d7c5b6",
    "12:36:27",
    "fix: Keep retained speech after field switch",
    SUPER503_AUTOMATION_NAMES.addressFeedback,
  ),
  partial: revision(
    "70dd13488e14ea2607752739a71af53012387813",
    "12:48:58",
    "fix: Keep pending words after partial finals",
    SUPER503_AUTOMATION_NAMES.addressFeedback,
  ),
  render: revision(
    "99951afa405afaaed98dbbb7a6bd646cf182a630",
    "13:00:34",
    "fix: Sync dictation snapshots after render",
    SUPER503_AUTOMATION_NAMES.addressFeedback,
  ),
};

function checksPhase(input: {
  runId: string;
  startedAt: string;
  duration: string;
  revision: SplitRunRevision;
  workflow: string;
}): SplitRunPhase {
  const sha = input.revision.sha ?? "";
  const shortSha = sha.slice(0, 7);
  return super503Phase({
    id: `pr-feedback-${input.runId}`,
    name: `Checks passed on [${shortSha}](https://github.com/superplanehq/superplane/commit/${sha})`,
    description: `· [ci/semaphoreci/pr: CI](https://superplanehq.semaphoreci.com/workflows/${input.workflow}): The build passed on Semaphore 2.0.`,
    componentName: SUPER503_AUTOMATION_NAMES.fixChecks,
    duration: input.duration,
    startedAt: input.startedAt,
    stream: super503ChecksStream({ prefix: `checks-${shortSha}`, at: input.startedAt, waitDuration: input.duration }),
    appId: SUPER503_APPS.fixChecks,
    runId: input.runId,
    revision: input.revision,
  });
}

/**
 * The deploy publishes its link as a task artifact once, on the first
 * run. Later runs report the same URL in their activity text only.
 */
function storybookPhase(input: {
  runId: string;
  startedAt: string;
  duration: string;
  revision: SplitRunRevision;
  addsLink?: boolean;
}): SplitRunPhase {
  return super503Phase({
    id: `pr-feedback-${input.runId}`,
    name: `Storybook deployment ready at ${SUPER503_STORYBOOK_URL}`,
    componentName: SUPER503_AUTOMATION_NAMES.storybook,
    duration: input.duration,
    startedAt: input.startedAt,
    stream: super503StorybookStream({
      prefix: `storybook-${input.revision.sha?.slice(0, 7)}`,
      at: input.startedAt,
      duration: input.duration,
      link: input.addsLink ? SUPER503_STORYBOOK_ARTIFACT : undefined,
    }),
    appId: SUPER503_APPS.storybook,
    runId: input.runId,
    costCents: "1",
    revision: input.revision,
  });
}

function reviewPhase(input: {
  runId: string;
  log: Super503ReviewLogKey;
  reviewId: string;
  description: string;
  startedAt: string;
  duration: string;
  agentDuration: string;
  costCents: string;
  totalTokens: string;
  revision: SplitRunRevision;
}): SplitRunPhase {
  return super503Phase({
    id: `pr-feedback-${input.runId}`,
    name: `${GREPTILE_REVIEW_TITLE}(https://github.com/superplanehq/superplane/pull/7771#pullrequestreview-${input.reviewId})`,
    description: input.description.trim(),
    componentName: SUPER503_AUTOMATION_NAMES.addressFeedback,
    duration: input.duration,
    startedAt: input.startedAt,
    stream: super503ReviewStream({
      prefix: input.log,
      at: input.startedAt,
      agentDuration: input.agentDuration,
      log: input.log,
    }),
    appId: SUPER503_APPS.addressFeedback,
    runId: input.runId,
    costCents: input.costCents,
    totalTokens: input.totalTokens,
    revision: input.revision,
  });
}

const MERGE_PHASE = super503Phase({
  id: `pr-feedback-${SUPER503_CLOSURE_RUN.runId}`,
  name: "PR #7771 was merged",
  componentName: SUPER503_AUTOMATION_NAMES.closure,
  duration: "2s",
  startedAt: "13:11:07",
  stream: super503ClosureStream(),
  ...SUPER503_CLOSURE_RUN,
  onPullRequest: true,
});

export const SUPER503_PULL_REQUEST_PHASES: SplitRunPhase[] = [
  checksPhase({
    runId: "eb5b062e-ec2a-45e4-9fe2-979fbc72b081",
    startedAt: "11:56:58",
    duration: "7m 16s",
    revision: SHA.initial,
    workflow: "4aecc7b6-0f97-4198-a973-052306832432?pipeline_id=c0c695e3-26d3-4f40-8ed8-9a7a6ad3d759",
  }),
  storybookPhase({
    runId: "26ca3895-2502-42a7-a567-376d0e6e31fa",
    startedAt: "11:56:58",
    duration: "1m 12s",
    revision: SHA.initial,
    addsLink: true,
  }),
  reviewPhase({
    runId: "56e9455a-0b43-40c7-b917-cbc46b331319",
    log: "review1",
    reviewId: "5290667624",
    description: review1,
    startedAt: "12:01:35",
    duration: "9m 32s",
    agentDuration: "9m 30s",
    costCents: "165",
    totalTokens: "1781630",
    revision: SHA.initial,
  }),
  checksPhase({
    runId: "ee0dae57-6a4b-4b30-843a-47a4c6bf9994",
    startedAt: "12:11:01",
    duration: "7m 21s",
    revision: SHA.focus,
    workflow: "66bfe97c-481f-4ee8-aa22-cfb31a695e35?pipeline_id=f25acb30-43b8-4506-b99f-d6db402a4578",
  }),
  storybookPhase({
    runId: "432223ff-1d8e-433a-a03e-9d583a567ac2",
    startedAt: "12:11:01",
    duration: "1m 21s",
    revision: SHA.focus,
  }),
  reviewPhase({
    runId: "4c68e5c0-07aa-4af0-93e5-ce4a80fc74c0",
    log: "review2",
    reviewId: "5290799354",
    description: review2,
    startedAt: "12:15:07",
    duration: "9m 11s",
    agentDuration: "9m 9s",
    costCents: "162",
    totalTokens: "2039100",
    revision: SHA.focus,
  }),
  checksPhase({
    runId: "8a6af718-f43b-4cfd-b2dc-8dd724c44aa0",
    startedAt: "12:24:13",
    duration: "6m 28s",
    revision: SHA.duplicate,
    workflow: "500285fb-05c2-425e-b9ec-2a6507da4fad?pipeline_id=6cb0574a-dbfe-4b11-90bf-7aa65799ad72",
  }),
  storybookPhase({
    runId: "4704d4a8-9c43-40e0-9b3b-f3bbbac272ea",
    startedAt: "12:24:13",
    duration: "1m 26s",
    revision: SHA.duplicate,
  }),
  reviewPhase({
    runId: "8db5e69f-4205-43d4-9320-377a1c14efa3",
    log: "review3",
    reviewId: "5290929541",
    description: review3,
    startedAt: "12:28:00",
    duration: "8m 33s",
    agentDuration: "8m 31s",
    costCents: "136",
    totalTokens: "1807569",
    revision: SHA.duplicate,
  }),
  checksPhase({
    runId: "f61c7742-4833-4c34-8f3d-e332bdbea750",
    startedAt: "12:36:27",
    duration: "8m 37s",
    revision: SHA.retained,
    workflow: "d4bdcff3-e364-4410-bcb2-0f8cc2219fb8?pipeline_id=e53f5d3a-f98a-4195-81d4-26cbedb6a966",
  }),
  storybookPhase({
    runId: "05cc479d-d095-4434-8c08-80b1a2bf958c",
    startedAt: "12:36:27",
    duration: "1m 18s",
    revision: SHA.retained,
  }),
  reviewPhase({
    runId: "84b46f90-3a0f-4c43-a9fe-9a9833068c5b",
    log: "review4",
    reviewId: "5291068956",
    description: review4,
    startedAt: "12:41:01",
    duration: "8m 6s",
    agentDuration: "8m 4s",
    costCents: "128",
    totalTokens: "1565814",
    revision: SHA.retained,
  }),
  checksPhase({
    runId: "a49940b1-6c51-4076-91c7-6660a008ecd2",
    startedAt: "12:48:58",
    duration: "8m 14s",
    revision: SHA.partial,
    workflow: "e15d6aa6-d785-43c3-b99b-127a365efadc?pipeline_id=da6a6467-79c9-4cb8-ad51-b2cabb429802",
  }),
  storybookPhase({
    runId: "dfe87e05-8243-40b4-be71-a880e78bdd0c",
    startedAt: "12:48:58",
    duration: "1m 21s",
    revision: SHA.partial,
  }),
  reviewPhase({
    runId: "b95f0016-1ec7-44f3-ad1f-a19bcc3e8031",
    log: "review5",
    reviewId: "5291222282",
    description: review5,
    startedAt: "12:53:59",
    duration: "6m 40s",
    agentDuration: "6m 38s",
    costCents: "113",
    totalTokens: "1534156",
    revision: SHA.partial,
  }),
  checksPhase({
    runId: "cb916cf0-a8d0-4bc6-85d3-6b22bff29107",
    startedAt: "13:00:34",
    duration: "7m 2s",
    revision: SHA.render,
    workflow: "e95c02df-d519-40d1-87b4-1effa718d345?pipeline_id=1eb7f4df-fe20-4b5e-943a-61e1c2684655",
  }),
  storybookPhase({
    runId: "8185fc64-a7b2-4f8b-9ec3-c04932799966",
    startedAt: "13:00:34",
    duration: "3m 56s",
    revision: SHA.render,
  }),
  MERGE_PHASE,
];
