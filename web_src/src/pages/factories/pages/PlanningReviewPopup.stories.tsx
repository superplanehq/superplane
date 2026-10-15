import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";

import { PlanningReviewPopup } from "./PlanningReviewPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { draftWithBashCommand, LONG_BASH_COMMAND, playFindBoxCloses } from "./planningReviewFindBox.storyHelpers";
import { RunOverlayBoardBackdrop } from "./work-order-popup-redesign/popupShared";

const AUTOMATION_HREF = "/organizations/demo-org/factories/refunds/apps/app-refund-planner?configure=1";

/**
 * Simple editing mode for one phase agent.
 */
const meta = {
  title: "Factories/Pages/Planning Review",
  component: PlanningReviewPopup,
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
} satisfies Meta<typeof PlanningReviewPopup>;

export default meta;

type Story = StoryObj<typeof PlanningReviewPopup>;

function popup(draft = PLANNING_REVIEW_DRAFT) {
  return (
    <MemoryRouter>
      <div className="relative min-h-svh">
        <RunOverlayBoardBackdrop />
        <PlanningReviewPopup onClose={() => undefined} initialDraft={draft} automationHref={AUTOMATION_HREF} />
      </div>
    </MemoryRouter>
  );
}

export const Mockup: Story = {
  name: "One agent",
  render: () => popup(),
};

export const OneLineFindBox: Story = {
  name: "One line find box",
  render: () => popup(draftWithBashCommand('git clone --depth 1 "$REPO_URL" repo')),
  play: playFindBoxCloses,
};

export const LongScriptFindBox: Story = {
  name: "Long script find box",
  render: () => popup(draftWithBashCommand(LONG_BASH_COMMAND)),
  play: playFindBoxCloses,
};
