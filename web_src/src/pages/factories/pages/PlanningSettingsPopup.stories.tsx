import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";

import { PlanningSettingsPopup } from "./PlanningSettingsPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { draftWithBashCommand, LONG_BASH_COMMAND } from "./planningReviewFindBox.storyHelpers";
import { DEFAULT_PLANNING_SETTINGS } from "./planningSettingsModel";

const meta = {
  title: "Factories/Pages/Planning Settings",
  component: PlanningSettingsPopup,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof PlanningSettingsPopup>;

export default meta;

type Story = StoryObj<typeof PlanningSettingsPopup>;

function popup(command?: string) {
  return (
    <MemoryRouter>
      <div className="relative min-h-svh bg-background">
        <PlanningSettingsPopup
          settings={DEFAULT_PLANNING_SETTINGS}
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
