import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { draftWithBashCommand, LONG_BASH_COMMAND } from "./planningReviewFindBox.storyHelpers";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";

const settings: PRFeedbackDraftSettings = {
  source: "discussion",
  name: "Address PR feedback",
  repository: "acme/payments",
  mention: "@superplaneagent",
  ignoreBots: true,
  allowedBots: [],
  checkNames: [],
  maximumAttempts: 3,
  runnerIntegrationIds: [],
};

const meta = {
  title: "Factories/Pages/PR Feedback Settings",
  component: PRFeedbackSettingsPopup,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof PRFeedbackSettingsPopup>;

export default meta;

type Story = StoryObj<typeof PRFeedbackSettingsPopup>;

function popup(command?: string) {
  return (
    <MemoryRouter>
      <div className="relative min-h-svh bg-background">
        <PRFeedbackSettingsPopup
          settings={settings}
          healthy
          onSave={() => undefined}
          onClose={() => undefined}
          fixed={false}
          initialTab="agent"
          agent={{
            draft: command ? draftWithBashCommand(command) : PLANNING_REVIEW_DRAFT,
            organizationId: "org-1",
            onSave: () => undefined,
          }}
        />
      </div>
    </MemoryRouter>
  );
}

export const AgentTab: Story = {
  name: "Agent tab",
  render: () => popup(),
};

export const AgentLongScript: Story = {
  name: "Agent long script",
  render: () => popup(LONG_BASH_COMMAND),
};
