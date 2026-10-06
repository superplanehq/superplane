import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { SPEC_ARTIFACT_NAME } from "../lib/intentDocument";
import { DRAFT_WORK_ORDER, PRIMARY_FACTORY_ID, type FactoriesFixture } from "./factoryPageResponses";
import { refineChatBoardFixture, refineChatPlanningSession } from "./refineChatBoardFixture";
import { clarityCheck } from "./workOrderCheckFixtures";

export type TaskPlanningReviewOptions = {
  confidence: 2 | 4;
  response: "questions" | "message" | "ready" | "updating";
  hasPlan?: boolean;
  clarity?: 2 | 5;
};

const TITLE = "Add customer logos to the homepage";
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
        `# ${TITLE}`,
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
  confidence,
  response,
  hasPlan = true,
  clarity,
}: TaskPlanningReviewOptions): FactoriesFixture {
  const fixture = refineChatBoardFixture();
  const orderId = DRAFT_WORK_ORDER.id!;
  const session = refineChatPlanningSession();
  const scoreAt = new Date(Date.now() - 120_000).toISOString();
  const summary =
    confidence === 2
      ? "Company evidence is incomplete. The agent needs help to confirm the shortlist and logo assets."
      : "The change is limited to one homepage section. The layout and verification steps are defined.";

  return {
    ...fixture,
    factories: fixture.factories.map((factory) => ({
      ...factory,
      planning: { ...factory.planning, enabled: true, confidence: true, clarity: clarity !== undefined },
    })),
    workOrdersByFactoryId: {
      ...fixture.workOrdersByFactoryId,
      [PRIMARY_FACTORY_ID]: fixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID].map((order) =>
        order.id === orderId ? { ...order, title: TITLE, description: DESCRIPTION } : order,
      ),
    },
    checksByOrderId: {
      ...fixture.checksByOrderId,
      [orderId]: [
        ...(clarity === undefined ? [] : [clarityCheck(orderId, clarity)]),
        {
          id: "check-homepage-confidence",
          key: "confidence",
          name: "Confidence score",
          score: confidence,
          maxScore: 5,
          level: confidence === 2 ? "LEVEL_CRITICAL" : "LEVEL_POSITIVE",
          summary,
          analysis: `## Evidence\n\n${summary}`,
          updatedAt: scoreAt,
        },
      ],
    },
    artifactsByOrderId: {
      ...fixture.artifactsByOrderId,
      [orderId]: hasPlan ? [planArtifact(response)] : [],
    },
    planningSessionsByWorkOrderId: {
      [orderId]: {
        ...session,
        draft: { ...session.draft, title: TITLE, description: DESCRIPTION },
        waitState: response === "updating" ? "" : "pending",
        messages: [
          { id: "plan-score", role: "plan", text: JSON.stringify({ score: clarity ?? 5 }), createdAt: scoreAt },
          {
            id: "agent-summary",
            role: "agent",
            text: hasPlan
              ? AGENT_MESSAGE[response]
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
                ],
              }
            : null,
      },
    },
  };
}
