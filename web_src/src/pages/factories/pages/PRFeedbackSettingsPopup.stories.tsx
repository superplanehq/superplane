import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";

const settings: PRFeedbackDraftSettings = {
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

const meta = {
  title: "Factories/Pages/PR Feedback Settings",
  component: PRFeedbackSettingsPopup,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof PRFeedbackSettingsPopup>;

export default meta;

type Story = StoryObj<typeof PRFeedbackSettingsPopup>;

function popup() {
  return (
    <MemoryRouter initialEntries={["/?prFeedbackSettings=agent"]}>
      <div className="relative min-h-svh bg-background">
        <PRFeedbackSettingsPopup
          settings={settings}
          onSave={(next) => console.log("save PR feedback settings", next)}
          onClose={() => console.log("close PR feedback settings")}
          fixed={false}
          initialTab="agent"
          agent={{
            draft: PLANNING_REVIEW_DRAFT,
            organizationId: "org-1",
            onSave: (draft) => console.log("save PR feedback agent", draft),
          }}
        />
      </div>
    </MemoryRouter>
  );
}

export const AddressPRFeedback: Story = {
  name: "Address PR feedback",
  render: () => popup(),
};
