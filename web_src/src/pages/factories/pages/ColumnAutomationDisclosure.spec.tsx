import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { useColumnAutomationCollapsePreference } from "@/hooks/useColumnAutomationCollapsePreference";
import type { ColumnAutomation } from "../lib/columnAutomations";
import { WorkOrderBoardLane } from "../workOrders/WorkOrderBoardChrome";
import { ColumnAutomationToggle } from "./ColumnAutomationDisclosure";
import { columnAutomationRowsSubheader } from "./columnAutomationRowsSubheader";

const automations: ColumnAutomation[] = [0, 1, 2].map((index) => ({
  id: `automation-${index}`,
  kind: "intake",
  name: `Automation ${index}`,
  trigger: "On event",
  action: "Create a task",
  iconSrc: "",
  iconAlt: "",
  health: index === 2 ? "disabled" : "healthy",
  runningCount: 0,
  catalogId: "github-issues",
}));

function Lane({
  identity = ["org", "workspace", "line", "backlog"],
  count = 2,
  title = "Backlog",
  names = true,
}: {
  identity?: string[];
  count?: number;
  title?: string;
  names?: boolean;
}) {
  const disclosure = useColumnAutomationCollapsePreference(JSON.stringify(identity), count);
  const list = automations.slice(0, count);
  return (
    <WorkOrderBoardLane
      title={title}
      count={1}
      emptyDescription="No tasks"
      subheaderOwnSpacing
      actions={names && count > 0 ? <ColumnAutomationToggle title={title} disclosure={disclosure} /> : undefined}
      subheader={columnAutomationRowsSubheader({
        title,
        automations: list,
        rowCount: names ? Math.max(1, count) : undefined,
        disclosure,
        onRowAction: () => undefined,
        onAdd: () => undefined,
        testId: `${title}-rows`,
      })}
    >
      <button type="button">Task in {title}</button>
    </WorkOrderBoardLane>
  );
}

beforeEach(() => window.localStorage.clear());
afterEach(() => vi.restoreAllMocks());

describe("column automation disclosure", () => {
  it("defaults from actual counts and toggles independently without replacing task content", async () => {
    const user = userEvent.setup();
    render(
      <>
        <Lane />
        <Lane title="Verify" identity={["org", "workspace", "line", "verify"]} count={3} />
      </>,
    );
    const backlog = screen.getByRole("list", { name: "Backlog automations" });
    const task = screen.getByRole("button", { name: "Task in Verify" });
    const expand = screen.getByRole("button", { name: "Expand automations for Verify" });
    expect(expand).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("list", { name: "Verify automations" })).not.toBeInTheDocument();
    expect(window.localStorage.length).toBe(0);
    expand.focus();
    await user.keyboard("{Enter}");
    expect(screen.getByRole("list", { name: "Verify automations" })).toBeVisible();
    expect(expand).toHaveAttribute(
      "aria-controls",
      screen.getByRole("list", { name: "Verify automations" }).closest("[id]")?.id,
    );
    await user.click(screen.getByRole("button", { name: "Collapse automations for Verify" }));
    expand.focus();
    await user.tab();
    expect(task).toHaveFocus();
    expect(screen.getByRole("list", { name: "Backlog automations" })).toBe(backlog);
    expect(screen.getByRole("button", { name: "Task in Verify" })).toBe(task);
  });

  it("retains both explicit states across counts, names, views, identity changes, and remounts", async () => {
    const user = userEvent.setup();
    const first = ["org", "workspace", "line", "backlog"];
    const view = render(<Lane identity={first} />);
    await user.click(screen.getByRole("button", { name: "Collapse automations for Backlog" }));
    for (const index of [0, 1, 2, 3]) {
      const other = first.map((part, position) => (position === index ? `${part}-other` : part));
      view.rerender(<Lane identity={other} count={3} />);
      await user.click(screen.getByRole("button", { name: "Expand automations for Backlog" }));
      view.rerender(<Lane identity={first} count={1} title="Renamed" />);
      expect(screen.getByRole("button", { name: "Expand automations for Renamed" })).toHaveAttribute(
        "aria-expanded",
        "false",
      );
      view.rerender(<Lane identity={other} count={3} />);
      expect(screen.getByRole("button", { name: "Collapse automations for Backlog" })).toHaveAttribute(
        "aria-expanded",
        "true",
      );
    }
    view.rerender(<Lane identity={first} names={false} />);
    expect(screen.queryByRole("button", { name: /automations for/ })).not.toBeInTheDocument();
    view.rerender(<Lane identity={first} />);
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    view.unmount();
    const expanded = render(<Lane identity={["org", "workspace", "line", "backlog-other"]} count={3} />);
    expect(screen.getByRole("button", { name: "Collapse automations for Backlog" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expanded.unmount();
    render(<Lane identity={first} count={1} />);
    expect(screen.getByRole("button", { name: "Expand automations for Backlog" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("follows count changes until the user makes an explicit choice", () => {
    const view = render(<Lane count={2} />);
    view.rerender(<Lane count={3} />);
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    view.rerender(<Lane count={1} />);
    expect(screen.getByRole("list")).toBeVisible();
    expect(window.localStorage.length).toBe(0);
  });

  it("uses count defaults for invalid storage and permits local toggling when storage fails", async () => {
    const user = userEvent.setup();
    vi.spyOn(window.localStorage, "getItem").mockReturnValue("invalid");
    const view = render(<Lane count={3} />);
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    view.unmount();
    vi.spyOn(window.localStorage, "getItem").mockImplementation(() => {
      throw new Error("Storage unavailable");
    });
    vi.spyOn(window.localStorage, "setItem").mockImplementation(() => {
      throw new Error("Storage unavailable");
    });
    render(<Lane count={3} />);
    await user.click(screen.getByRole("button", { name: "Expand automations for Backlog" }));
    expect(screen.getByRole("list")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Collapse automations for Backlog" }));
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });

  it("keeps the empty add row after a collapsed list becomes empty", async () => {
    const user = userEvent.setup();
    const view = render(<Lane count={3} />);
    view.rerender(<Lane count={0} />);
    expect(screen.queryByRole("button", { name: /automations for/ })).not.toBeInTheDocument();
    await user.click(within(screen.getByRole("list")).getByRole("button", { name: "Add automation for Backlog" }));
  });
});
