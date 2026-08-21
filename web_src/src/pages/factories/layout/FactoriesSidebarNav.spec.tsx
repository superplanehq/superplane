import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { FACTORIES_ORGANIZATION_ID, REFUND_FACTORY, REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageResponses";
import { factoryHomePath, factorySettingsPath, factoryVelocityPath, workOrdersPath } from "../lib/factoryPagePaths";
import { FactoriesSidebarNav } from "./FactoriesSidebarNav";

function renderNav(path: string, onCreateWorkOrder = vi.fn()) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <TooltipProvider>
        <FactoriesSidebarNav
          organizationId={FACTORIES_ORGANIZATION_ID}
          factoryKey={REFUND_FACTORY.key!}
          lineId={REFUND_LINE_PLAN_ID}
          canOpenSettings
          canCreateWorkOrder
          permissionsLoading={false}
          onCreateWorkOrder={onCreateWorkOrder}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

const org = FACTORIES_ORGANIZATION_ID;
const key = REFUND_FACTORY.key!;

describe("FactoriesSidebarNav", () => {
  it("places Intake, Board, Velocity, Settings, and Create under the switcher", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`);

    const nav = screen.getByTestId("factories-sidebar-nav");
    const controls = [
      screen.getByTestId("factories-nav-intake"),
      screen.getByTestId("factories-nav-board"),
      screen.getByTestId("factories-nav-velocity"),
      screen.getByTestId("factories-workspace-settings-link"),
      screen.getByTestId("factories-sidebar-create-work-order"),
    ];

    expect(controls.map((node) => nav.contains(node))).toEqual([true, true, true, true, true]);
    expect(screen.getByTestId("factories-nav-intake")).toHaveAttribute("href", workOrdersPath(org, key));
    expect(screen.getByTestId("factories-nav-board")).toHaveAttribute(
      "href",
      factoryHomePath(org, key, REFUND_LINE_PLAN_ID),
    );
    expect(screen.getByTestId("factories-nav-velocity")).toHaveAttribute("href", factoryVelocityPath(org, key));
    expect(screen.getByTestId("factories-workspace-settings-link")).toHaveAttribute(
      "href",
      factorySettingsPath(org, key),
    );
  });

  it("marks the Board icon current on the line board", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`);

    expect(screen.getByTestId("factories-nav-board")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("factories-nav-intake")).not.toHaveAttribute("aria-current");
  });

  it("marks the Intake icon current on work orders", () => {
    renderNav(`/${org}/workspaces/${key}/work-orders`);

    expect(screen.getByTestId("factories-nav-intake")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("factories-nav-board")).not.toHaveAttribute("aria-current");
  });

  it("opens create work order from the plus control", async () => {
    const onCreateWorkOrder = vi.fn();
    const user = userEvent.setup();
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`, onCreateWorkOrder);

    await user.click(screen.getByTestId("factories-sidebar-create-work-order"));
    expect(onCreateWorkOrder).toHaveBeenCalledTimes(1);
  });
});
