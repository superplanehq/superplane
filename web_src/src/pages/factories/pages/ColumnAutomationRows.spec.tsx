import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationRows } from "./ColumnAutomationRows";

const GITHUB_INTAKE: ColumnAutomation = {
  id: "intake-github",
  kind: "intake",
  name: "GitHub issues",
  trigger: "On GitHub issue",
  action: "Create a task",
  iconSrc: "",
  iconAlt: "GitHub",
  health: "healthy",
  runningCount: 0,
  catalogId: "github-issues",
};

const ANALYSIS: ColumnAutomation = {
  ...GITHUB_INTAKE,
  id: "analysis",
  kind: "analysis",
  name: "Task analysis",
  catalogId: "analysis",
};

describe("ColumnAutomationRows", () => {
  it("lists one sentence per automation", () => {
    render(<ColumnAutomationRows title="Backlog" automations={[GITHUB_INTAKE, ANALYSIS]} rowCount={2} testId="rows" />);

    const list = screen.getByRole("list", { name: "Backlog automations" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows.map((row) => row.textContent)).toEqual(["Listens to GitHub issues", "Scores new tasks"]);
  });

  it("pads short columns with blank slots so every header keeps the same height", () => {
    render(<ColumnAutomationRows title="Done" automations={[GITHUB_INTAKE]} rowCount={3} testId="rows" />);

    expect(screen.getAllByRole("listitem")).toHaveLength(1);
    expect(screen.getAllByTestId("rows-blank")).toHaveLength(2);
  });

  it("names the empty state when the column has no automation", () => {
    render(<ColumnAutomationRows title="Review" automations={[]} rowCount={2} testId="rows" />);

    expect(screen.getByTestId("rows-empty")).toHaveTextContent("No automations");
    expect(screen.queryByRole("button", { name: "Add automation for Review" })).not.toBeInTheDocument();
    expect(screen.getAllByTestId("rows-blank")).toHaveLength(1);
  });

  it("offers Add automation when the empty row can open a picker", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn();
    render(<ColumnAutomationRows title="Review" automations={[]} rowCount={1} onAdd={onAdd} testId="rows" />);

    await user.click(screen.getByRole("button", { name: "Add automation for Review" }));
    expect(onAdd).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("rows-empty")).toHaveTextContent("Add automation");
  });

  it("marks an automation that needs repair", () => {
    render(
      <ColumnAutomationRows
        title="Backlog"
        automations={[{ ...GITHUB_INTAKE, health: "needs-repair" }]}
        rowCount={1}
        testId="rows"
      />,
    );

    expect(screen.getByTestId("rows-needs-repair-intake-github")).toBeInTheDocument();
  });

  it("opens the automation when the row is clicked", async () => {
    const user = userEvent.setup();
    const onRowAction = vi.fn();
    render(
      <ColumnAutomationRows
        title="Backlog"
        automations={[GITHUB_INTAKE]}
        rowCount={1}
        onRowAction={onRowAction}
        testId="rows"
      />,
    );

    await user.click(screen.getByTestId("rows-row-intake-github"));
    expect(onRowAction).toHaveBeenCalledWith(GITHUB_INTAKE, "settings");
    expect(screen.queryByTestId("rows-menu-intake-github")).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    expect(screen.getByTestId("rows-settings-intake-github")).toBeInTheDocument();
    expect(screen.getByTestId("rows-row-intake-github").className).toContain("hover:bg-black/10");
  });

  it("hides the settings icon when the row cannot open settings", () => {
    render(<ColumnAutomationRows title="Backlog" automations={[GITHUB_INTAKE]} rowCount={1} testId="rows" />);

    expect(screen.queryByTestId("rows-settings-intake-github")).not.toBeInTheDocument();
  });

  it("does not draw a box around the rows", () => {
    render(<ColumnAutomationRows title="Backlog" automations={[GITHUB_INTAKE]} rowCount={1} testId="rows" />);

    expect(screen.getByTestId("rows").className).not.toMatch(/border|rounded-md|bg-background/);
  });
});
