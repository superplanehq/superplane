import { FEATURE_TASK_PLANNING_REVIEW } from "@/lib/experimentalFeatures";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { refundLineCanvasFixture } from "../../__fixtures__/factoryOwnedCanvasFixture";
import { DRAFT_WORK_ORDER, PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "../../__fixtures__/factoryPageResponses";
import { refineChatPlanningSession } from "../../__fixtures__/refineChatBoardFixture";
import {
  taskPlanningReviewFixture,
  type TaskPlanningReviewOptions,
} from "../../__fixtures__/taskPlanningReviewFixture";

const planningRunId = refineChatPlanningSession().canvasRunId!;
const baseCanvasFixture = refundLineCanvasFixture();
const planningCanvasFixture = {
  ...baseCanvasFixture,
  runDetailsById: {
    ...baseCanvasFixture.runDetailsById,
    // The default canvas run has finished. Keep this planning session live so
    // the updating story does not immediately fall back to its previous score.
    [planningRunId]: { run: { id: planningRunId, state: "STATE_STARTED" } },
  },
};

const meta = {
  title: "Factories/Pages/Task Planning",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
    docs: {
      description: {
        component:
          "Current task popup for issues #7834 and #7837. Compare pending questions, a free-text request, and a ready plan. The planningReviewEnabled control switches the organization feature flag to compare the original and new UI.",
      },
    },
  },
  args: { planningReviewEnabled: true },
  argTypes: {
    planningReviewEnabled: {
      control: "boolean",
      description: "Enable Task Planning Review for the fixture organization",
    },
  },
  render: (args) => (
    <FactoriesHarness
      key={JSON.stringify(args)}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/${DRAFT_WORK_ORDER.number}?lineId=${REFUND_FACTORY_LINES[0].id}`}
      factoriesFixture={taskPlanningReviewFixture(args)}
      appFixture={planningCanvasFixture}
      experimentalFeatures={args.planningReviewEnabled ? [FEATURE_TASK_PLANNING_REVIEW] : []}
    />
  ),
} satisfies Meta<TaskPlanningReviewOptions & { planningReviewEnabled: boolean }>;

export default meta;
type Story = StoryObj<typeof meta>;

export const LowConfidenceQuestions: Story = {
  name: "Confidence 2 — pending questions and plan",
  args: { confidence: 2, response: "questions" },
};

export const LowConfidenceQuestionsWithoutPlan: Story = {
  name: "Confidence 2 — pending questions, no plan",
  args: { confidence: 2, response: "questions", hasPlan: false },
};

export const LowConfidenceSummary: Story = {
  name: "Confidence 2 — summary and plan",
  args: { confidence: 2, response: "message" },
};

export const ReadyPlan: Story = {
  name: "Confidence 4 — ready plan",
  args: { confidence: 4, response: "ready" },
};

export const HighConfidenceQuestions: Story = {
  name: "Confidence 4 — pending questions",
  args: { confidence: 4, response: "questions" },
};

export const LowClarityQuestions: Story = {
  name: "Clarity 2, confidence 4 — pending questions",
  args: { confidence: 4, clarity: 2, response: "questions" },
};

export const UpdatingPlan: Story = {
  name: "Updating a plan with previous scores",
  args: { confidence: 2, response: "updating" },
};

export const OriginalReadyPlan: Story = {
  name: "Original UI — ready plan (flag off)",
  args: { confidence: 4, response: "ready", planningReviewEnabled: false },
};

export const OriginalLowConfidenceQuestions: Story = {
  name: "Original UI — pending questions (flag off)",
  args: { confidence: 2, response: "questions", planningReviewEnabled: false },
};
