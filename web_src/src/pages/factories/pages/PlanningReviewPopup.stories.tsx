import type { Meta, StoryObj } from "@storybook/react-vite";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";

import { PermissionsContext } from "@/contexts/permissionsContextState";

import { PlanningReviewPopup } from "./PlanningReviewPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { RunOverlayBoardBackdrop } from "./work-order-popup-redesign/popupShared";

const AUTOMATION_HREF = "/organizations/demo-org/factories/refunds/apps/app-refund-planner?configure=1";

function withEditorPermissions(story: ReactNode) {
  return (
    <PermissionsContext.Provider
      value={{
        permissions: [],
        isLoading: false,
        canAct: () => true,
        currentUserId: "storybook-user",
        browserNotificationPreferences: { enabled: false, showWhileViewing: true },
      }}
    >
      {story}
    </PermissionsContext.Provider>
  );
}

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

export const Mockup: Story = {
  name: "One agent",
  render: () => (
    <MemoryRouter>
      <div className="relative min-h-svh">
        <RunOverlayBoardBackdrop />
        {withEditorPermissions(
          <PlanningReviewPopup
            onClose={() => undefined}
            initialDraft={PLANNING_REVIEW_DRAFT}
            automationHref={AUTOMATION_HREF}
          />,
        )}
      </div>
    </MemoryRouter>
  ),
};
