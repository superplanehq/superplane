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
        component: "Task popup for pending questions, a free-text request, and a ready plan.",
      },
    },
  },
  render: (args) => (
    <FactoriesHarness
      key={JSON.stringify(args)}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/${DRAFT_WORK_ORDER.number}?lineId=${REFUND_FACTORY_LINES[0].id}`}
      factoriesFixture={taskPlanningReviewFixture(args)}
      appFixture={planningCanvasFixture}
    />
  ),
} satisfies Meta<TaskPlanningReviewOptions>;

export default meta;
type Story = StoryObj<typeof meta>;

export const LowConfidenceQuestions: Story = {
  name: "Two checks need review · questions and plan",
  args: { scores: { clarity: 2, complexity: 3, verifiability: 2 }, response: "questions" },
};

export const SingleQuestionWithSkip: Story = {
  name: "One question · Skip available",
  args: { scores: { clarity: 2, complexity: 3, verifiability: 2 }, response: "questions", singleQuestion: true },
};

export const LowConfidenceQuestionsWithoutPlan: Story = {
  name: "Two checks need review · questions, no plan",
  args: { scores: { clarity: 2, complexity: 3, verifiability: 2 }, response: "questions", hasPlan: false },
};

export const LowConfidenceSummary: Story = {
  name: "Two checks need review · summary and plan",
  args: { scores: { clarity: 2, complexity: 3, verifiability: 2 }, response: "message" },
};

export const LowClaritySummary: Story = {
  name: "Clarity critical · one review warning",
  args: { scores: { clarity: 1, complexity: 3, verifiability: 2 }, response: "message" },
};

export const ReadyPlan: Story = {
  name: "All checks ready · ready plan",
  args: { scores: { clarity: 3, complexity: 3, verifiability: 3 }, response: "ready" },
};

export const ModerateComplexityQuestions: Story = {
  name: "Complexity needs review · questions",
  args: { scores: { clarity: 3, complexity: 2, verifiability: 3 }, response: "questions" },
};

export const LowClarityQuestions: Story = {
  name: "Clarity critical · unclear company choice",
  args: { scores: { clarity: 1, complexity: 3, verifiability: 2 }, response: "questions" },
};

export const UpdatingPlan: Story = {
  name: "Updating a plan with previous scores",
  args: { scores: { clarity: 2, complexity: 3, verifiability: 2 }, response: "updating" },
};

export const CreatingPlanWithoutPlan: Story = {
  name: "Creating the first plan · no Start",
  args: { scores: { clarity: 2, complexity: 3, verifiability: 2 }, response: "updating", hasPlan: false },
};
