import type { FactoriesWorkOrderArtifact, FactoriesWorkOrderCheck } from "@/api-client";

import {
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  confidenceCheckLevel,
  confidenceSuitabilityAnalysis,
  confidenceSuitabilitySummary,
} from "../lib/confidenceScore";
import { SPEC_ARTIFACT_NAME } from "../lib/intentDocument";
import type { PlanningSessionPayload } from "../pages/planningSessionView";
import { DEFAULT_ARTIFACTS_BY_ORDER_ID } from "./factoryPageEventFixtures";
import { DRAFT_WORK_ORDER, STORYBOOK_ME_USER_ID, type FactoriesFixture } from "./factoryPageResponses";
import { lineMetricsFactoriesFixture } from "./lineMetricsFactoriesFixture";
import { clarityCheck } from "./workOrderCheckFixtures";

const SESSION_ID = "ps-draft-refunds";
/** Clarity: the storage path is still open. */
const PLAN_SCORE = 2;
/** Confidence: the header change is small once the path is decided. */
const CONFIDENCE_SCORE = 4;

function minutesAgo(minutes: number): string {
  return new Date(Date.now() - minutes * 60_000).toISOString();
}

const LEVEL_FOR_CHECK: Record<ReturnType<typeof confidenceCheckLevel>, FactoriesWorkOrderCheck["level"]> = {
  positive: "LEVEL_POSITIVE",
  neutral: "LEVEL_NEUTRAL",
  caution: "LEVEL_CAUTION",
  critical: "LEVEL_CRITICAL",
};

function draftConfidenceCheck(): FactoriesWorkOrderCheck {
  const band = confidenceBandForScore(CONFIDENCE_SCORE);
  return {
    id: "check-confidence-wo-draft-refunds",
    key: "confidence",
    name: CONFIDENCE_CHECK_NAME,
    score: CONFIDENCE_SCORE,
    maxScore: CONFIDENCE_SCORE_MAX,
    level: LEVEL_FOR_CHECK[confidenceCheckLevel(CONFIDENCE_SCORE)],
    summary: confidenceSuitabilitySummary(band),
    analysis: confidenceSuitabilityAnalysis({
      source: "GitHub",
      reasons: [
        "The first change is limited to the task header and a similar control exists to copy.",
        "The reaction store has tests the agent can extend.",
        "The score is not 5 because the storage path needs one human look.",
      ],
    }),
    automation: { appId: "app-line-refine", appName: "Refine Task" },
    runId: "run-refine-wo-draft-refunds",
    updatedAt: minutesAgo(6),
  };
}

function draftPlanArtifact(): FactoriesWorkOrderArtifact {
  return {
    id: "art-spec-wo-draft-refunds",
    type: "TYPE_MARKDOWN",
    data: {
      name: SPEC_ARTIFACT_NAME,
      title: SPEC_ARTIFACT_NAME,
      body: [
        "# Task reactions",
        "",
        "## Executive summary",
        "",
        "Add emoji reactions on the task header. Do not add a picker on list cards in this change.",
        "",
        "## Plan",
        "",
        "- Add a reaction control on the task header.",
        "- Store one reaction per user and emoji.",
        "- A user can remove only their own reaction.",
      ].join("\n"),
    },
  };
}

export function refineChatPlanningSession(): PlanningSessionPayload {
  return {
    id: SESSION_ID,
    factoryId: lineMetricsFactoriesFixture.factories[0]?.id,
    repository: "acme/payments",
    state: "running",
    waitState: "pending",
    canvasId: "canvas-refine-draft-refunds",
    canvasRunId: "run-refine-draft-refunds",
    executionId: "exec-refine-draft-refunds",
    selectableModelKey: "hosted::anthropic::claude-sonnet-4-6",
    kind: "PLANNING_SESSION_KIND_WORK_ORDER_ANALYSIS",
    draft: {
      title: DRAFT_WORK_ORDER.title,
      description: DRAFT_WORK_ORDER.description,
      workOrderId: DRAFT_WORK_ORDER.id,
    },
    messages: [
      {
        id: "agent-read",
        role: "agent",
        text: "I read the request. People need a reaction on the task itself, not only on comments.",
        createdAt: minutesAgo(18),
      },
      {
        id: "user-scope",
        role: "user",
        userId: STORYBOOK_ME_USER_ID,
        text: "Keep this on the task header. Do not add a reaction picker on every list card.",
        createdAt: minutesAgo(16),
      },
      {
        id: "agent-survey-ask",
        role: "agent",
        text: "I updated the plan. The first version is too wide. I need one answer before I can raise the score.",
        createdAt: minutesAgo(14),
      },
      {
        id: "user-survey-reply",
        role: "user",
        userId: STORYBOOK_ME_USER_ID,
        text: "Where do people add a reaction? Task header only\nWho can remove a reaction? The person who added it",
        createdAt: minutesAgo(12),
      },
      {
        id: "plan-score",
        role: "plan",
        text: JSON.stringify({ score: PLAN_SCORE }),
        createdAt: minutesAgo(8),
      },
      {
        id: "agent-follow-up",
        role: "agent",
        text: "I limited the first change to the task header. The score stays low because the storage path is still open.",
        createdAt: minutesAgo(7),
      },
    ],
    survey: {
      id: "survey-draft-refunds",
      questions: [
        {
          prompt: "What should the first change include?",
          options: [
            "Reactions on the task header only",
            "Reactions on cards and the header",
            "Reactions in search results",
          ],
        },
        {
          prompt: "How should SuperPlane store reactions?",
          options: ["New table", "Add a column on the task", "Reuse comment reactions"],
        },
      ],
    },
  };
}

/** Line board with the draft refine popup already in a mid-session state. */
export function refineChatBoardFixture(): FactoriesFixture {
  const orderId = DRAFT_WORK_ORDER.id ?? "wo-draft-refunds";
  return {
    ...lineMetricsFactoriesFixture,
    checksByOrderId: {
      ...lineMetricsFactoriesFixture.checksByOrderId,
      [orderId]: [clarityCheck(orderId, PLAN_SCORE, 6), draftConfidenceCheck()],
    },
    artifactsByOrderId: {
      ...lineMetricsFactoriesFixture.artifactsByOrderId,
      [orderId]: [...(DEFAULT_ARTIFACTS_BY_ORDER_ID[orderId] ?? []), draftPlanArtifact()],
    },
    planningSessionsByWorkOrderId: {
      [orderId]: refineChatPlanningSession(),
    },
  };
}
