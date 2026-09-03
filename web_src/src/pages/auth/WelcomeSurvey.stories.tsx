import type { Meta, StoryObj } from "@storybook/react-vite";

import { withFactoriesTheme } from "@/pages/factories/__fixtures__/factoriesStoryTheme";

import PostHogSurveyForm from "./PostHogSurveyForm";
import { WelcomeSurveyLayout } from "./WelcomeSurveyLayout";
import { onboardingSurvey } from "./welcomeSurveyStoryFixtures";

/**
 * The `/welcome` survey after the theme-factories restyle.
 * Choice questions render as a lettered poll. Dark stories show the same
 * screens with the Theme toolbar set to dark.
 */
const meta = {
  title: "Auth/Welcome survey",
  component: PostHogSurveyForm,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "The onboarding survey on /welcome. Choice questions use lettered poll rows. Open text uses the Describe the task placeholder when PostHog does not set one.",
      },
      story: {
        inline: false,
        iframeHeight: 900,
      },
    },
  },
  tags: ["autodocs"],
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <WelcomeSurveyLayout>
        <Story />
      </WelcomeSurveyLayout>
    ),
  ],
  args: {
    survey: onboardingSurvey,
    redirectTo: "/",
    onComplete: () => {
      // eslint-disable-next-line no-console
      console.log("Survey completed");
    },
  },
} satisfies Meta<typeof PostHogSurveyForm>;

export default meta;

type Story = StoryObj<typeof meta>;

const darkGlobals = { theme: "dark" as const, backgrounds: { value: "dark" } };

/** Full three-question survey. Select an option or Skip to move to the next step. */
export const Journey: Story = {
  name: "0 Clickable journey",
};

/** Question 1 of 3. Single-choice poll with lettered rows. */
export const HowYouHeard: Story = {
  name: "1 How you heard",
  args: { initialQuestionIndex: 0 },
};

export const HowYouHeardDark: Story = {
  name: "1b How you heard (dark)",
  globals: darkGlobals,
  args: { initialQuestionIndex: 0 },
};

/** Question 2 of 3. Multiple-choice poll keeps the checkbox and a Continue button. */
export const MultipleChoice: Story = {
  name: "2 Multiple choice",
  args: { initialQuestionIndex: 1 },
};

export const MultipleChoiceDark: Story = {
  name: "2b Multiple choice (dark)",
  globals: darkGlobals,
  args: { initialQuestionIndex: 1 },
};

/** Question 3 of 3. Open text with the new agent-task copy. */
export const AgentTask: Story = {
  name: "3 Agent task",
  args: { initialQuestionIndex: 2 },
};

export const AgentTaskDark: Story = {
  name: "3b Agent task (dark)",
  globals: darkGlobals,
  args: { initialQuestionIndex: 2 },
};
