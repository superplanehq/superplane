import type { FactoriesWorkOrderArtifact, FactoriesWorkOrderCheck } from "@/api-client";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";

import { presentWorkOrderChecks } from "../../lib/workOrderChecks";
import { doneFooterForStatus } from "./splitRunFooter";
import type { SplitRunFixture, SplitRunPhase } from "./splitRunMocks";
import { SUPER503_PULL_REQUEST_PHASES } from "./splitRunSuper503PullRequest";
import {
  SUPER503_APPS,
  SUPER503_AUTOMATION_NAMES,
  SUPER503_CLOSURE_RUN,
  SUPER503_IMPLEMENT_RUN,
  SUPER503_PULL_REQUEST,
  SUPER503_REPOSITORY,
  super503Phase,
} from "./splitRunSuper503Shared";
import {
  SUPER503_MODEL,
  super503AnalysisStream,
  super503BacklogStream,
  super503ImplementStream,
  super503Time,
} from "./splitRunSuper503Streams";

export { SUPER503_IMPLEMENT_RUN, super503Phase } from "./splitRunSuper503Shared";

/**
 * Storybook fixture from SUPER-503, "Dictation papercuts", on the
 * Superplane Steam Machine line. Every value comes from the production
 * work order: the Backlog automation scored it and wrote `spec.md`,
 * Implement opened PR #7771, five Greptile rounds followed, and PR
 * Closure closed the task when the pull request merged.
 *
 * Production reports no spend for the Backlog run and no model for pull
 * request activity, so those cards carry none here either.
 */

const BRANCH = "fix/put-dictation-in-input";
const TITLE = "Dictation papercuts";
const ANALYSIS_RUN_ID = "f6c3162b-9311-4ecb-81ce-78a0906cbb6e";

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

const SPEC_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-503-spec",
  type: "TYPE_MARKDOWN",
  data: { name: "spec.md", title: "spec.md", body: PLAN_TEXT },
};

const BRANCH_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-503-branch",
  type: "TYPE_BRANCH",
  data: {
    name: BRANCH,
    repository: SUPER503_REPOSITORY,
    url: `https://github.com/${SUPER503_REPOSITORY}/tree/${BRANCH}`,
  },
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

const CONFIDENCE_CHECK: FactoriesWorkOrderCheck = {
  id: "check-503-confidence",
  key: "confidence",
  name: "Confidence score",
  score: 4,
  maxScore: 5,
  level: "LEVEL_POSITIVE",
  summary:
    "An agent can drop the extra line and reuse the existing speech helpers in one run. A quick listen in the form is still worth it after.",
  previousScore: 4,
  recentScores: [4, 4],
  runId: ANALYSIS_RUN_ID,
  updatedAt: super503Time("11:33:11"),
};

const ANALYSIS_CHECKS = presentWorkOrderChecks([CONFIDENCE_CHECK]);

export const SUPER503_BACKLOG_PHASE = super503Phase({
  id: "backlog",
  name: "Backlog",
  componentName: "Created manually",
  duration: "2s",
  startedAt: "11:28:53",
  artifacts: [DESCRIPTION_ARTIFACT],
  stream: super503BacklogStream(DESCRIPTION_ARTIFACT),
});

/**
 * The Backlog automation's run: 11:28:54 until the owner started the task
 * at 11:37:09. The agent itself worked 4m 8s across two chat turns.
 */
export const SUPER503_ANALYSIS_PHASE = super503Phase({
  id: "analysis",
  name: "Analysis",
  componentName: SUPER503_AUTOMATION_NAMES.backlog,
  duration: "8m 15s",
  startedAt: "11:28:54",
  artifacts: [SPEC_ARTIFACT],
  checks: ANALYSIS_CHECKS,
  stream: super503AnalysisStream(SPEC_ARTIFACT),
  appId: SUPER503_APPS.backlog,
  runId: ANALYSIS_RUN_ID,
});

const IMPLEMENT_PHASE = super503Phase({
  id: "implement",
  name: "Implement",
  componentName: SUPER503_AUTOMATION_NAMES.implement,
  duration: "19m 50s",
  startedAt: "11:37:08",
  artifacts: [BRANCH_ARTIFACT, EVIDENCE_IMAGE, EVIDENCE_VIDEO],
  stream: super503ImplementStream(BRANCH_ARTIFACT, SUPER503_PULL_REQUEST),
  ...SUPER503_IMPLEMENT_RUN,
  stepIndex: 0,
  costCents: "733",
  totalTokens: "11482898",
  model: SUPER503_MODEL,
});

const TASK_PHASES: SplitRunPhase[] = [SUPER503_BACKLOG_PHASE, SUPER503_ANALYSIS_PHASE, IMPLEMENT_PHASE];

export const SPLIT_RUN_SUPER503: SplitRunFixture = {
  title: TITLE,
  descriptionText: DESCRIPTION_TEXT,
  owner: OWNER,
  assigneeIds: [OWNER.id],
  elapsed: "1h 42m",
  startedLabel: "Started yesterday",
  costUsd: "$15.07",
  tokensLabel: "20.7M tokens",
  usageByModel: [{ provider: "openrouter", model: "x-ai/grok-4.6", totalTokens: "20700858", costCents: "1507" }],
  lineName: "Superplane Steam Machine",
  currentStepIndex: 0,
  lineStatus: "passed",
  currentPhaseId: "done",
  openPhaseId: "implement",
  phases: [...TASK_PHASES, ...SUPER503_PULL_REQUEST_PHASES],
  waitingNotes: [],
  checks: ANALYSIS_CHECKS,
  footer: {
    ...doneFooterForStatus("completed", { automationName: SUPER503_AUTOMATION_NAMES.closure }),
    run: SUPER503_CLOSURE_RUN,
  },
  footerTone: "done",
};
