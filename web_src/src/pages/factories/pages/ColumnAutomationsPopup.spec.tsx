import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";

import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationsHeaderSlot } from "./ColumnAutomationsIndicator";
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

const AGENT: ColumnAutomation = {
  id: "step-0-app-refund-implementer",
  kind: "agent-step",
  name: "Implement",
  trigger: "On task in Implement",
  action: "Run the Implement agent",
  iconSrc: "",
  iconAlt: "",
  health: "healthy",
  runningCount: 0,
  catalogId: "agent-step",
  canvasId: "app-refund-implementer",
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
  return render(<ColumnAutomationsPopup automation={INTAKE} onAction={vi.fn()} defaultOpen {...props} />);
}

describe("ColumnAutomationsPopup", () => {
  it("opens a summary from the automation icon", async () => {
    const user = userEvent.setup();
    render(<ColumnAutomationsPopup automation={INTAKE} onAction={vi.fn()} />);

    expect(screen.queryByTestId("column-automations-popup")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("column-automation-icon-intake-github"));

    expect(screen.getByTestId("column-automations-popup")).toBeInTheDocument();
    expect(screen.getByTestId("column-automation-row-intake-github")).toHaveTextContent(
      "On GitHub issue → Create a task in Backlog",
    );
    expect(screen.queryByTestId("column-automations-add")).not.toBeInTheDocument();
  });

  it("shows needs-repair and disabled badges", () => {
    renderPopup({ automation: REPAIR });
    expect(screen.getByTestId("column-automation-row-intake-sentry")).toHaveTextContent("Needs repair");
    expect(screen.getByTestId("column-automation-icon-intake-sentry-needs-repair")).toBeInTheDocument();

    renderPopup({ automation: DISABLED });
    expect(screen.getByTestId("column-automation-row-analysis-1")).toHaveTextContent("Disabled");
  });

  it("does not show a running count", () => {
    renderPopup({ automation: { ...INTAKE, runningCount: 4 } });

    expect(screen.getByTestId("column-automation-row-intake-github")).not.toHaveTextContent("running");
  });

  it("shows last-run activity when it is supplied", () => {
    renderPopup({
      automation: { ...INTAKE, name: "Task analysis" },
      activity: {
        lastRunStatus: "passed",
        lastRunWhen: "2 minutes ago",
        runningCount: 2,
      },
    });

    const activity = screen.getByTestId("column-automation-activity");
    expect(activity).toHaveTextContent("Passed");
    expect(activity).toHaveTextContent("2 minutes ago");
    expect(activity).toHaveTextContent("2 running");
    expect(activity.querySelector("svg.animate-spin")).not.toBeNull();
    expect(screen.queryByTestId("column-automation-hourly-chart")).not.toBeInTheDocument();
  });

  it("reports a click on the summary", async () => {
    const user = userEvent.setup();
    const onAction = vi.fn();
    renderPopup({ onAction });

    await user.click(screen.getByTestId("column-automation-row-intake-github"));
    expect(onAction).toHaveBeenCalledWith("settings");
  });

  it("does not offer Edit Agent or Edit Automation", () => {
    renderPopup({ automation: AGENT });

    expect(screen.queryByRole("menuitem", { name: "Edit Agent" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Edit Automation" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-step-0-app-refund-implementer-edit-agent")).not.toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-step-0-app-refund-implementer-edit")).not.toBeInTheDocument();
  });
});

describe("ColumnAutomationsHeaderSlot", () => {
  it("renders one icon for each automation", () => {
    render(
      <ColumnAutomationsHeaderSlot
        title="Backlog"
        automations={[INTAKE, DISABLED]}
        onRowAction={vi.fn()}
        testId="lines-backlog-automations"
      />,
    );

    expect(screen.getByTestId("lines-backlog-automations")).toBeInTheDocument();
    expect(screen.getByTestId("column-automation-icon-intake-github")).toHaveAttribute("aria-label", "GitHub issues");
    expect(screen.getByTestId("column-automation-icon-analysis-1")).toHaveAttribute("aria-label", "Task analysis");
    expect(screen.getByTestId("lines-backlog-automations")).not.toHaveTextContent("2");
  });

  it("hides the strip when the column has no automations", () => {
    const { container } = render(
      <ColumnAutomationsHeaderSlot
        title="Review"
        automations={[]}
        onRowAction={vi.fn()}
        testId="lines-phase-3-automations"
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });
});
