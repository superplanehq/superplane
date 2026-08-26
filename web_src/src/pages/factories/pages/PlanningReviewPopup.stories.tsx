import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";

import { PlanningReviewPopup } from "./PlanningReviewPopup";
import { PLANNING_REVIEW_DRAFT, PLANNING_REVIEW_SINGLE_AGENT_DRAFT } from "./planningReviewMockup";
import { RunOverlayBoardBackdrop } from "./work-order-popup-redesign/popupShared";

const AUTOMATION_HREF = "/organizations/demo-org/factories/refunds/apps/app-refund-planner?configure=1";

/**
 * Simple editing mode for a phase agent. Column menu action Edit Agent
 * opens this popup.
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
  name: "Several agents",
  render: () => (
    <MemoryRouter>
      <div className="relative min-h-svh">
        <RunOverlayBoardBackdrop />
        <PlanningReviewPopup
          onClose={() => undefined}
          initialDraft={PLANNING_REVIEW_DRAFT}
          automationHref={AUTOMATION_HREF}
        />
      </div>
    </MemoryRouter>
  ),
};

export const SingleAgent: Story = {
  name: "One agent",
  render: () => (
    <MemoryRouter>
      <div className="relative min-h-svh">
        <RunOverlayBoardBackdrop />
        <PlanningReviewPopup
          onClose={() => undefined}
          initialDraft={PLANNING_REVIEW_SINGLE_AGENT_DRAFT}
          automationHref={AUTOMATION_HREF}
        />
      </div>
    </MemoryRouter>
  ),
};
