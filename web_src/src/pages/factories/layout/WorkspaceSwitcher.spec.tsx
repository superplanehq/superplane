import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import {
  ACME_ONBOARDING_FACTORY,
  FACTORIES_ORGANIZATION_ID,
  REFUND_FACTORY,
} from "../__fixtures__/factoryPageResponses";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

function renderSwitcher() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <WorkspaceSwitcher
          organizationId={FACTORIES_ORGANIZATION_ID}
          factory={REFUND_FACTORY}
          factories={[REFUND_FACTORY, ACME_ONBOARDING_FACTORY]}
          canCreateFactory
          permissionsLoading={false}
          onCreateFactory={() => undefined}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("WorkspaceSwitcher", () => {
  it("opens the workspace menu from the initials control", async () => {
    const user = userEvent.setup();
    renderSwitcher();

    const trigger = screen.getByTestId("factories-workspace-switch");
    expect(trigger).toHaveAccessibleName(/Switch workspace, Semaphore/);
    await user.click(trigger);
    expect(screen.getByTestId(`factories-workspace-option-${REFUND_FACTORY.id}`)).toHaveTextContent("Semaphore");
    expect(screen.getByTestId(`factories-workspace-option-${ACME_ONBOARDING_FACTORY.id}`)).toHaveTextContent(
      "Acme onboarding",
    );
  });
});
