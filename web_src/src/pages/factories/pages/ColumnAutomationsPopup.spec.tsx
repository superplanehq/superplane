import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";

import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationsPopup } from "./ColumnAutomationsPopup";

const INTAKE: ColumnAutomation = {
  id: "intake-github",
  kind: "intake",
  name: "GitHub issues",
  trigger: "On GitHub issue",
  action: "Create a task in Backlog",
  iconSrc: "",
  iconAlt: "GitHub",
  health: "healthy",
  runningCount: 0,
  catalogId: "github-issues",
  canvasId: "app-github",
};

const REPAIR: ColumnAutomation = {
  ...INTAKE,
  id: "intake-sentry",
  name: "Sentry exceptions",
  trigger: "On Sentry exception",
  catalogId: "sentry-exceptions",
  health: "needs-repair",
};

const DISABLED: ColumnAutomation = {
  ...INTAKE,
  id: "analysis-1",
  kind: "analysis",
  name: "Task analysis",
  trigger: "On task in Backlog",
  action: "Score the task",
  catalogId: "analysis",
  health: "disabled",
};

function renderPopup(props: Partial<ComponentProps<typeof ColumnAutomationsPopup>> = {}) {
  return render(
    <ColumnAutomationsPopup
      columnTitle="Backlog"
      columnKey="backlog"
      automations={[INTAKE]}
      open
      onOpen={vi.fn()}
      onClose={vi.fn()}
      onAdd={vi.fn()}
      onRowAction={vi.fn()}
      trigger={<button type="button">Open automations</button>}
      {...props}
    />,
  );
}

describe("ColumnAutomationsPopup", () => {
  it("lists automations with a trigger-to-action sentence", () => {
    renderPopup();

    expect(screen.getByRole("heading", { name: "Backlog automations" })).toBeInTheDocument();
    expect(screen.getByTestId("column-automation-row-intake-github")).toHaveTextContent(
      "On GitHub issue → Create a task in Backlog",
    );
    expect(screen.getByTestId("column-automations-divider")).toBeInTheDocument();
    expect(screen.getByTestId("column-automations-add")).toHaveTextContent("New automation");
    expect(screen.getByTestId("column-automations-add")).toHaveTextContent("Create a canvas and open the editor.");
    expect(screen.queryByRole("button", { name: "GitHub issues menu" })).not.toBeInTheDocument();
  });

  it("shows the empty copy for a phase with no automations", () => {
    renderPopup({ columnTitle: "Review", columnKey: "phase-1", automations: [] });

    expect(screen.getByTestId("column-automations-empty")).toHaveTextContent(
      "No automations. Tasks pass through Review without action.",
    );
  });

  it("shows needs-repair and disabled badges", () => {
    renderPopup({ automations: [REPAIR, DISABLED] });

    expect(screen.getByTestId("column-automation-row-intake-sentry")).toHaveTextContent("Needs repair");
    expect(screen.getByTestId("column-automation-row-analysis-1")).toHaveTextContent("Disabled");
  });

  it("does not show a running count", () => {
    renderPopup({ automations: [{ ...INTAKE, runningCount: 4 }] });

    expect(screen.getByTestId("column-automation-row-intake-github")).not.toHaveTextContent("running");
  });

  it("reports row click and New automation", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn();
    const onRowAction = vi.fn();
    renderPopup({ onAdd, onRowAction });

    await user.click(screen.getByTestId("column-automation-row-intake-github"));
    expect(onRowAction).toHaveBeenCalledWith(INTAKE, "settings");

    await user.click(screen.getByTestId("column-automations-add"));
    expect(onAdd).toHaveBeenCalledTimes(1);
  });
});
