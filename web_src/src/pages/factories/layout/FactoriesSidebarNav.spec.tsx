import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { FACTORIES_ORGANIZATION_ID, REFUND_FACTORY, REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageResponses";
import { factoryHomePath, factorySettingsPath, factoryVelocityPath } from "../lib/factoryPagePaths";
import { FactoriesSidebarNav } from "./FactoriesSidebarNav";

function renderNav(path: string, showVelocity = false) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <TooltipProvider>
        <FactoriesSidebarNav
          organizationId={FACTORIES_ORGANIZATION_ID}
          factoryKey={REFUND_FACTORY.key!}
          lineId={REFUND_LINE_PLAN_ID}
          canOpenSettings
          permissionsLoading={false}
          showVelocity={showVelocity}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

const org = FACTORIES_ORGANIZATION_ID;
const key = REFUND_FACTORY.key!;

describe("FactoriesSidebarNav", () => {
  it("places Board and Settings under the switcher", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`);

    const nav = screen.getByTestId("factories-sidebar-nav");
    const controls = [
      screen.getByTestId("factories-nav-board"),
      screen.getByTestId("factories-workspace-settings-link"),
    ];

    expect(controls.map((node) => nav.contains(node))).toEqual([true, true]);
    expect(screen.queryByTestId("factories-sidebar-create-work-order")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-nav-velocity")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-nav-intake")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-nav-pr-feedback")).not.toBeInTheDocument();
    expect(screen.getByTestId("factories-nav-board")).toHaveAttribute(
      "href",
      factoryHomePath(org, key, REFUND_LINE_PLAN_ID),
    );
    expect(screen.getByTestId("factories-workspace-settings-link")).toHaveAttribute(
      "href",
      factorySettingsPath(org, key),
    );
  });

  it("marks the Board icon current on the line board", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`);

    expect(screen.getByTestId("factories-nav-board")).toHaveAttribute("aria-current", "page");
  });

  it("keeps the Board icon current while a listener shows its settings", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);
    expect(screen.getByTestId("factories-nav-board")).toHaveAttribute("aria-current", "page");

    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}?prFeedback=1`);
    expect(screen.getAllByTestId("factories-nav-board")[1]).toHaveAttribute("aria-current", "page");
  });

  it("shows the Velocity link when showVelocity is true", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`, true);

    const nav = screen.getByTestId("factories-sidebar-nav");
    const velocityLink = screen.getByTestId("factories-nav-velocity");

    expect(nav.contains(velocityLink)).toBe(true);
    expect(velocityLink).toHaveAttribute("href", factoryVelocityPath(org, key));
  });

  it("marks the Velocity icon current on the velocity page", () => {
    renderNav(`/${org}/workspaces/${key}/velocity`, true);

    expect(screen.getByTestId("factories-nav-velocity")).toHaveAttribute("aria-current", "page");
  });
});
