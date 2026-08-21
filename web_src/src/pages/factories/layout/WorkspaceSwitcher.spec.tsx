import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { FACTORIES_ORGANIZATION_ID, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
import { factorySettingsPath } from "../lib/factoryPagePaths";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

function renderSwitcher() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <WorkspaceSwitcher
          organizationId={FACTORIES_ORGANIZATION_ID}
          factory={REFUND_FACTORY}
          factories={[REFUND_FACTORY]}
          canOpenSettings
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
  });

  it("links workspace settings from the rail", () => {
    renderSwitcher();

    const settingsLink = screen.getByTestId("factories-workspace-settings-link");
    expect(settingsLink).toHaveAttribute("href", factorySettingsPath(FACTORIES_ORGANIZATION_ID, REFUND_FACTORY.key!));
  });
});
