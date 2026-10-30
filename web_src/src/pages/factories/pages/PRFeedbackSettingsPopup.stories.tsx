import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import type { PlanningReviewDraft } from "./planningReviewMockup";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";

const discussionSettings: PRFeedbackDraftSettings = {
  source: "discussion",
  name: "Address PR feedback",
  repository: "acme/payments",
  mention: "@superplaneagent",
  ignoreBots: true,
  addressAllBots: false,
  allowedBots: [],
  checkNames: [],
  maximumAttempts: 3,
  runnerIntegrationIds: [],
};

const checksSettings: PRFeedbackDraftSettings = {
  source: "checks",
  name: "Fix pull request checks",
  repository: "acme/app",
  mention: "",
  ignoreBots: false,
  addressAllBots: false,
  allowedBots: [],
  checkNames: ["lint", "unit"],
  maximumAttempts: 3,
  runnerIntegrationIds: [],
};

const discussionAgentDraft: PlanningReviewDraft = {
  title: "Address PR feedback",
  components: [
    {
      id: "address-pr-feedback",
      title: "Address PR feedback",
      description: "Address human and bot comments on the pull request.",
      expanded: true,
      component: "runnerSuperPlane",
      configuration: {
        model: "hosted::openrouter::anthropic/claude-sonnet-4-6",
        steps: [
          {
            name: "Address PR feedback",
            type: "prompt",
            prompt: "Read every new pull request comment and make the smallest change that resolves it.",
          },
        ],
        executionTimeoutSeconds: 1200,
      },
      concurrency: { max: "1", key: "pr-feedback-{{ $.data.pullRequest.number }}" },
    },
  ],
};

const checksAgentDraft: PlanningReviewDraft = {
  title: "Fix pull request checks",
  components: [
    {
      id: "fix-failing-checks",
      title: "Fix failing checks",
      description: "Read the failing check logs and push a fix to the pull request branch.",
      expanded: true,
      component: "runnerSuperPlane",
      configuration: {
        model: "hosted::openrouter::anthropic/claude-sonnet-4-6",
        steps: [
          {
            name: "Fix failing checks",
            type: "prompt",
            prompt: "Read the logs of the failing checks, fix the cause, and run the same check locally.",
          },
        ],
        executionTimeoutSeconds: 1800,
      },
      concurrency: { max: "1", key: "checks-fix-{{ $.data.pullRequest.number }}" },
    },
  ],
};

const meta = {
  title: "Factories/Pages/PR Feedback Settings",
  component: PRFeedbackSettingsPopup,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof PRFeedbackSettingsPopup>;

export default meta;

type Story = StoryObj<typeof PRFeedbackSettingsPopup>;

function popup(settings: PRFeedbackDraftSettings, draft: PlanningReviewDraft) {
  return (
    <MemoryRouter>
      <div className="relative min-h-svh bg-background">
        <PRFeedbackSettingsPopup
          settings={settings}
          onSave={(next) => console.log("save PR feedback settings", next)}
          onClose={() => console.log("close PR feedback settings")}
          fixed={false}
          agent={{
            draft,
            organizationId: "org-1",
            onSave: (next) => console.log("save PR feedback agent", next),
          }}
        />
      </div>
    </MemoryRouter>
  );
}

export const AddressPRFeedback: Story = {
  name: "Address PR feedback",
  render: () => popup(discussionSettings, discussionAgentDraft),
};

export const FixPullRequestChecks: Story = {
  name: "Fix pull request checks",
  render: () => popup(checksSettings, checksAgentDraft),
};
