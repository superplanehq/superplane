import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";
import { FACTORIES_ORGANIZATION_ID, REFUND_FACTORY, REFUND_LINE_PLAN_ID } from "../__fixtures__/factoryPageResponses";
import { factoryAgentPath, factoryHomePath, factoryVelocityPath } from "../lib/factoryPagePaths";
import { FactoriesSidebarNav } from "./FactoriesSidebarNav";

function renderNav(path: string, { showAgent = false }: { showAgent?: boolean } = {}) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <TooltipProvider>
        <FactoriesSidebarNav
          organizationId={FACTORIES_ORGANIZATION_ID}
          factoryKey={REFUND_FACTORY.key!}
          lineId={REFUND_LINE_PLAN_ID}
          showAgent={showAgent}
        />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

const org = FACTORIES_ORGANIZATION_ID;
const key = REFUND_FACTORY.key!;

describe("FactoriesSidebarNav", () => {
  it("places Board and Velocity under the switcher", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`);

    const nav = screen.getByTestId("factories-sidebar-nav");
    const controls = [screen.getByTestId("factories-nav-board"), screen.getByTestId("factories-nav-velocity")];

    expect(controls.map((node) => nav.contains(node))).toEqual([true, true]);
    expect(screen.queryByTestId("factories-workspace-settings-link")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-sidebar-create-work-order")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-nav-intake")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-nav-pr-feedback")).not.toBeInTheDocument();
    expect(screen.getByTestId("factories-nav-board")).toHaveAttribute(
      "href",
      factoryHomePath(org, key, REFUND_LINE_PLAN_ID),
    );
    expect(screen.getByTestId("factories-nav-board").querySelector(".lucide-house")).toBeInTheDocument();
    expect(screen.getByTestId("factories-nav-velocity").querySelector(".lucide-chart-line")).toBeInTheDocument();
    expect(screen.queryByTestId("factories-nav-agent")).not.toBeInTheDocument();
  });

  it("places Agent between Board and Velocity", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`, { showAgent: true });

    const links = [...screen.getByTestId("factories-sidebar-nav").querySelectorAll("a")].map((node) =>
      node.getAttribute("data-testid"),
    );
    const agentLink = screen.getByTestId("factories-nav-agent");

    expect(links).toEqual(["factories-nav-board", "factories-nav-agent", "factories-nav-velocity"]);
    expect(agentLink).toHaveAttribute("aria-label", "Agent");
    expect(agentLink).toHaveAttribute("href", factoryAgentPath(org, key));
    expect(agentLink.querySelector(".lucide-cpu")).toBeInTheDocument();
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

  it("shows the Velocity link", () => {
    renderNav(`/${org}/workspaces/${key}/lines/${REFUND_LINE_PLAN_ID}`);

    const nav = screen.getByTestId("factories-sidebar-nav");
    const velocityLink = screen.getByTestId("factories-nav-velocity");

    expect(nav.contains(velocityLink)).toBe(true);
    expect(velocityLink).toHaveAttribute("href", factoryVelocityPath(org, key));
  });

  it("marks the Velocity icon current on the velocity page", () => {
    renderNav(`/${org}/workspaces/${key}/velocity`);

    expect(screen.getByTestId("factories-nav-velocity")).toHaveAttribute("aria-current", "page");
  });

  it("marks the Agent icon current on the agent page", () => {
    renderNav(`/${org}/workspaces/${key}/agent`, { showAgent: true });

    expect(screen.getByTestId("factories-nav-agent")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("factories-nav-board")).not.toHaveAttribute("aria-current");
    expect(screen.getByTestId("factories-nav-velocity")).not.toHaveAttribute("aria-current");
  });

  it("does not mark Agent current on agent-resources", () => {
    renderNav(`/${org}/workspaces/${key}/settings/workspace/agent-resources`, { showAgent: true });

    expect(screen.getByTestId("factories-nav-agent")).not.toHaveAttribute("aria-current");
  });
});
