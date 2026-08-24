import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { FACTORIES_ORGANIZATION_ID, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
import { factoryHomePath } from "../lib/factoryPagePaths";
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
  it("links the workspace name to the Home page", () => {
    renderSwitcher();

    const nameLink = screen.getByTestId("factories-workspace-home-link");
    expect(nameLink).toHaveTextContent("Semaphore");
    expect(nameLink).toHaveAttribute("href", factoryHomePath(FACTORIES_ORGANIZATION_ID, REFUND_FACTORY.key!));
  });
});
