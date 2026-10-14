import type { FactoriesWorkOrderArtifact, FactoriesWorkOrderCheck } from "@/api-client";

import { SPEC_ARTIFACT_NAME } from "../lib/intentDocument";
import { PLANNING_REVIEW_SCORE_MAX } from "../lib/planningReviewScore";
import { DRAFT_WORK_ORDER, PRIMARY_FACTORY_ID, type FactoriesFixture } from "./factoryPageResponses";
import { refineChatBoardFixture, refineChatPlanningSession } from "./refineChatBoardFixture";

export type TaskPlanningReviewOptions = {
  scores: { clarity: 1 | 2 | 3; complexity: 1 | 2 | 3; verifiability: 1 | 2 | 3 };
  response: "questions" | "message" | "ready" | "updating";
  hasPlan?: boolean;
  singleQuestion?: boolean;
};

type ReviewScore = TaskPlanningReviewOptions["scores"]["clarity"];

const REVIEW_SUMMARIES: Record<keyof TaskPlanningReviewOptions["scores"], Record<ReviewScore, string>> = {
  clarity: {
    1: "The homepage companies and logo assets are not specified.",
    2: "The company shortlist and logo assets need a decision.",
    3: "The plan names the approved companies and logo assets.",
  },
  complexity: {
    1: "The implementation depends on several unknown systems.",
    2: "The logo assets and responsive layout need coordination.",
    3: "The change is limited to one homepage section.",
  },
  verifiability: {
    1: "The plan has no reliable way to confirm the selected companies.",
    2: "The company evidence and approved assets still need confirmation.",
    3: "The plan checks company approval, labels, and layout.",
  },
};

function reviewCheck(
  key: keyof TaskPlanningReviewOptions["scores"],
  score: ReviewScore,
  updatedAt: string,
): FactoriesWorkOrderCheck {
  const summary = REVIEW_SUMMARIES[key][score];
  return {
    id: `check-homepage-${key}`,
    key,
    name: key.charAt(0).toUpperCase() + key.slice(1),
    score,
    maxScore: PLANNING_REVIEW_SCORE_MAX,
    level: score === 3 ? "LEVEL_POSITIVE" : score === 2 ? "LEVEL_CAUTION" : "LEVEL_CRITICAL",
    summary,
    analysis: `## Evidence\n\n${summary}`,
    updatedAt,
  };
}

const TASK_TITLE = "Review repository stars for company logos";
const PLAN_TITLE = "Add customer logos to the homepage";
const DESCRIPTION =
  "Review the repository stars and suggest companies for the homepage. Rank recognizable brands and teams with at least 10 people.";

function planArtifact(response: TaskPlanningReviewOptions["response"]): FactoriesWorkOrderArtifact {
  return {
    id: "art-homepage-plan",
    type: "TYPE_MARKDOWN",
    data: {
      name: SPEC_ARTIFACT_NAME,
      title: SPEC_ARTIFACT_NAME,
      body: [
        `# ${PLAN_TITLE}`,
        "",
        "## Executive summary",
        "",
        "Add a customer section below the homepage introduction. Use confirmed company names and approved logo assets.",
        "",
        "## Company shortlist",
        "",
        "| Company | Evidence | Rank |",
        "| --- | --- | --- |",
        "| Northstar | Recognizable brand; company listed on public profiles | 1 |",
        "| Fieldwork | Team of more than 10 people; company listed on public profiles | 2 |",
        "| Relay | Company listed on public profiles; team size unverified | 3 |",
        "",
        "Company names and evidence in this Storybook example are fictional.",
        "",
        "## Decisions",
        "",
        response === "ready"
          ? "Use Northstar and Fieldwork. Approved logo assets are available in the repository."
          : "Confirm the companies and approved logo assets before implementation.",
        "",
        "## Implementation",
        "",
        "1. Add the company section below the homepage introduction.",
        "2. Use approved logos with accessible company names.",
        "3. Keep the layout readable on narrow screens.",
        "",
        "## Verification",
        "",
        "Confirm that only approved companies appear. Check the logo labels and layout at desktop and mobile widths.",
      ].join("\n"),
    },
  };
}

const AGENT_MESSAGE: Record<TaskPlanningReviewOptions["response"], string> = {
  questions:
    "I prepared a first draft with a ranked company shortlist. I need two decisions before I can finish the plan.",
  message:
    "Most people on the stars list left company blank. I ranked known brands and teams of 10 or more. Reply with the companies you want on the homepage. This is a first pass.",
  ready:
    "I updated the plan to use Northstar and Fieldwork. The approved logo assets are available. The plan is ready to review.",
  updating: "I will update the plan with your selected companies and check the logo assets.",
};

/** Current production UI, seeded for issues #7834 and #7837 and adjacent states. */
export function taskPlanningReviewFixture({
  scores,
  response,
  hasPlan = true,
  singleQuestion = false,
}: TaskPlanningReviewOptions): FactoriesFixture {
  const fixture = refineChatBoardFixture();
  const orderId = DRAFT_WORK_ORDER.id!;
  const session = refineChatPlanningSession();
  const scoreAt = new Date(Date.now() - 120_000).toISOString();
  const confidence = Math.min(scores.clarity, scores.complexity, scores.verifiability);
  const agentMessage =
    singleQuestion && response === "questions"
      ? "I prepared a first draft with a ranked company shortlist. Which companies should appear on the homepage?"
      : AGENT_MESSAGE[response];

  return {
    ...fixture,
    factories: fixture.factories.map((factory) => ({
      ...factory,
      planning: { ...factory.planning, enabled: true, confidence: true },
    })),
    workOrdersByFactoryId: {
      ...fixture.workOrdersByFactoryId,
      [PRIMARY_FACTORY_ID]: fixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID].map((order) =>
        order.id === orderId ? { ...order, title: TASK_TITLE, description: DESCRIPTION } : order,
      ),
    },
    checksByOrderId: {
      ...fixture.checksByOrderId,
      [orderId]: [
        reviewCheck("clarity", scores.clarity, scoreAt),
        reviewCheck("complexity", scores.complexity, scoreAt),
        reviewCheck("verifiability", scores.verifiability, scoreAt),
      ],
    },
    artifactsByOrderId: {
      ...fixture.artifactsByOrderId,
      [orderId]: hasPlan ? [planArtifact(response)] : [],
    },
    planningSessionsByWorkOrderId: {
      [orderId]: {
        ...session,
        draft: { ...session.draft, title: TASK_TITLE, description: DESCRIPTION },
        waitState: response === "updating" ? "" : "pending",
        messages: [
          { id: "plan-score", role: "plan", text: JSON.stringify({ score: confidence }), createdAt: scoreAt },
          {
            id: "agent-summary",
            role: "agent",
            text: hasPlan
              ? agentMessage
              : response === "updating"
                ? "I am writing the first plan from the company research."
                : "I researched Northstar, Fieldwork, and Relay. I need two decisions before I can write the plan.",
            createdAt: scoreAt,
          },
        ],
        survey:
          response === "questions"
            ? {
                id: "survey-homepage-companies",
                questions: [
                  {
                    prompt: "Which companies should appear on the homepage?",
                    options: [
                      "Northstar and Fieldwork",
                      "Northstar, Fieldwork, and Relay",
                      "Research more companies first",
                    ],
                  },
                  {
                    prompt: "Which logo assets should the agent use?",
                    options: ["Approved assets in the repository", "Text names for now", "I will provide the assets"],
                  },
                ].slice(0, singleQuestion ? 1 : 2),
              }
            : null,
      },
    },
  };
}
