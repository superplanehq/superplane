import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";
import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_LINE_ID,
  FACTORIES_ORGANIZATION_ID,
  REFUND_FACTORY,
} from "../__fixtures__/factoryPageResponses";
import { factoryHomePath, factorySettingsWorkspaceGeneralPath, factoryVelocityPath } from "../lib/factoryPagePaths";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

function LocationProbe() {
  const location = useLocation();
  return <span data-testid="location-path">{`${location.pathname}${location.search}`}</span>;
}

function renderSwitcher(path = "/", { canOpenSettings = true }: { canOpenSettings?: boolean } = {}) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <TooltipProvider>
        <Routes>
          <Route
            path="*"
            element={
              <>
                <WorkspaceSwitcher
                  organizationId={FACTORIES_ORGANIZATION_ID}
                  factory={REFUND_FACTORY}
                  factories={[REFUND_FACTORY, ACME_ONBOARDING_FACTORY]}
                  canCreateFactory
                  canOpenSettings={canOpenSettings}
                  permissionsLoading={false}
                  onCreateFactory={() => undefined}
                />
                <LocationProbe />
              </>
            }
          />
        </Routes>
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
    expect(
      screen.getByTestId(`factories-workspace-option-${REFUND_FACTORY.id}`).querySelector("svg.lucide-check"),
    ).not.toBeNull();
    expect(
      screen.getByTestId(`factories-workspace-option-${ACME_ONBOARDING_FACTORY.id}`).querySelector("svg.lucide-check"),
    ).toBeNull();
    expect(screen.getByTestId("factories-workspace-settings-link")).toHaveAttribute(
      "href",
      factorySettingsWorkspaceGeneralPath(FACTORIES_ORGANIZATION_ID, REFUND_FACTORY.key!),
    );
    expect(screen.getByTestId(`factories-workspace-settings-${ACME_ONBOARDING_FACTORY.id}`)).toHaveAttribute(
      "href",
      factorySettingsWorkspaceGeneralPath(FACTORIES_ORGANIZATION_ID, ACME_ONBOARDING_FACTORY.key!),
    );
  });

  it("opens workspace settings from the cog without switching the board", async () => {
    const user = userEvent.setup();
    renderSwitcher(`/${FACTORIES_ORGANIZATION_ID}/workspaces/${REFUND_FACTORY.key}/lines/line-plan`);

    await user.click(screen.getByTestId("factories-workspace-switch"));
    await user.click(screen.getByTestId("factories-workspace-settings-link"));

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factorySettingsWorkspaceGeneralPath(FACTORIES_ORGANIZATION_ID, REFUND_FACTORY.key!),
    );
  });

  it("opens settings for another workspace from its cog", async () => {
    const user = userEvent.setup();
    renderSwitcher();

    await user.click(screen.getByTestId("factories-workspace-switch"));
    await user.click(screen.getByTestId(`factories-workspace-settings-${ACME_ONBOARDING_FACTORY.id}`));

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factorySettingsWorkspaceGeneralPath(FACTORIES_ORGANIZATION_ID, ACME_ONBOARDING_FACTORY.key!),
    );
  });

  it("hides workspace settings cogs when the user cannot open them", async () => {
    const user = userEvent.setup();
    renderSwitcher("/", { canOpenSettings: false });

    await user.click(screen.getByTestId("factories-workspace-switch"));
    expect(screen.queryByTestId("factories-workspace-settings-link")).not.toBeInTheDocument();
    expect(screen.queryByTestId(`factories-workspace-settings-${ACME_ONBOARDING_FACTORY.id}`)).not.toBeInTheDocument();
  });

  it("opens the other workspace on its line board without intake query", async () => {
    const user = userEvent.setup();
    renderSwitcher(`/${FACTORIES_ORGANIZATION_ID}/workspaces/${REFUND_FACTORY.key}/lines/line-plan?intake=1`);

    await user.click(screen.getByTestId("factories-workspace-switch"));
    await user.click(screen.getByTestId(`factories-workspace-option-${ACME_ONBOARDING_FACTORY.id}`));

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factoryHomePath(FACTORIES_ORGANIZATION_ID, ACME_ONBOARDING_FACTORY.key!, ACME_ONBOARDING_LINE_ID),
    );
  });

  it("keeps the current settings page in the other workspace", async () => {
    const user = userEvent.setup();
    renderSwitcher(factorySettingsWorkspaceGeneralPath(FACTORIES_ORGANIZATION_ID, REFUND_FACTORY.key!));

    await user.click(screen.getByTestId("factories-workspace-switch"));
    await user.click(screen.getByTestId(`factories-workspace-option-${ACME_ONBOARDING_FACTORY.id}`));

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factorySettingsWorkspaceGeneralPath(FACTORIES_ORGANIZATION_ID, ACME_ONBOARDING_FACTORY.key!),
    );
  });

  it("keeps Velocity in the other workspace", async () => {
    const user = userEvent.setup();
    renderSwitcher(factoryVelocityPath(FACTORIES_ORGANIZATION_ID, REFUND_FACTORY.key!));

    await user.click(screen.getByTestId("factories-workspace-switch"));
    await user.click(screen.getByTestId(`factories-workspace-option-${ACME_ONBOARDING_FACTORY.id}`));

    expect(screen.getByTestId("location-path")).toHaveTextContent(
      factoryVelocityPath(FACTORIES_ORGANIZATION_ID, ACME_ONBOARDING_FACTORY.key!),
    );
  });
});
