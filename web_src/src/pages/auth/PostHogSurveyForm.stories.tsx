import type { Meta, StoryObj } from "@storybook/react-vite";
import { withFactoriesTheme } from "@/pages/factories/__fixtures__/factoriesStoryTheme";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";

/**
 * Renders the questions of the "New User Onboarding Survey" fetched from
 * PostHog. Choice questions use the same poll look as the AI agent's
 * `SurveyWidget`; use the Storybook toolbar to preview light and dark mode.
 */
const meta = {
  title: "Auth/PostHogSurveyForm",
  component: PostHogSurveyForm,
  parameters: { layout: "padded" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <div className="mx-auto max-w-xl rounded-lg border border-border bg-card px-8 py-9 text-foreground shadow-sm">
        <Story />
      </div>
    ),
  ],
  args: {
    redirectTo: "/",
    onComplete: () => {
      // eslint-disable-next-line no-console
      console.log("Survey completed");
    },
  },
} satisfies Meta<typeof PostHogSurveyForm>;

export default meta;

type Story = StoryObj<typeof meta>;

const singleChoiceSurvey: PostHogSurvey = {
  id: "survey-single-choice",
  name: "New User Onboarding Survey",
  questions: [
    {
      id: "q-role",
      question: "What best describes your role?",
      type: "single_choice",
      choices: [
        "Software engineer (writes and ships code)",
        "Engineering manager (leads a team)",
        "Product manager (owns the roadmap)",
        "Other",
      ],
    },
  ],
};

const multipleChoiceSurvey: PostHogSurvey = {
  id: "survey-multiple-choice",
  name: "New User Onboarding Survey",
  questions: [
    {
      id: "q-tools",
      question: "Which tools does your team use today?",
      type: "multiple_choice",
      allow_multiple: true,
      choices: [
        "GitHub (source control)",
        "GitHub Actions (CI/CD)",
        "PagerDuty (incidents)",
        "Terraform (infrastructure)",
      ],
    },
  ],
};

const textSurvey: PostHogSurvey = {
  id: "survey-text",
  name: "New User Onboarding Survey",
  questions: [
    {
      id: "q-agent-task",
      question:
        "If you had a single task for an AI agent on your software development process today, what would it be?",
      type: "open",
      placeholder: "Describe the task",
    },
  ],
};

/** Single-choice question renders as a poll: lettered rows that advance on click. */
export const SingleChoiceQuestion: Story = {
  args: { survey: singleChoiceSurvey },
};

/** Multiple-choice question keeps the checkbox for multi-select, plus a Continue button. */
export const MultipleChoiceQuestion: Story = {
  args: { survey: multipleChoiceSurvey },
};

/** Open text question — the final question of the refreshed survey. */
export const TextQuestion: Story = {
  args: { survey: textSurvey },
};
