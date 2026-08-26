import type { Meta, StoryObj } from "@storybook/react-vite";

import { PlanningReviewPopup } from "./PlanningReviewPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { RunOverlayBoardBackdrop } from "./work-order-popup-redesign/popupShared";

/**
 * Simple editing mode for a phase agent. Column menu action Edit Automation
 * opens this popup instead of the canvas.
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

export const Mockup: Story = {
  name: "Edit automation",
  render: () => (
    <div className="relative min-h-svh">
      <RunOverlayBoardBackdrop />
      <PlanningReviewPopup onClose={() => undefined} initialDraft={PLANNING_REVIEW_DRAFT} />
    </div>
  ),
};

export const Empty: Story = {
  name: "New automation",
  render: () => (
    <div className="relative min-h-svh">
      <RunOverlayBoardBackdrop />
      <PlanningReviewPopup
        onClose={() => undefined}
        initialDraft={{
          ...PLANNING_REVIEW_DRAFT,
          title: "New automation",
        }}
      />
    </div>
  ),
};
