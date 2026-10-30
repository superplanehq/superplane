import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrderEvent } from "@/api-client";
import type { AutomationStage, ConsoleAutomation } from "./automationsViewModel";
import { stepActionRows } from "./consoleStepActions";
import type { StepActionRow } from "./consoleStepActions";
import type { SplitRunPhase } from "../splitRunMocks";

const shown: { on: boolean; rows: StepActionRow[]; error: boolean } = { on: false, rows: [], error: false };

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => shown.on && id === "factory_step_actions",
    enabledExperimentalFeatures: [],
    isLoading: false,
    organizationReady: true,
  }),
}));

vi.mock("./useStepActions", () => ({
  useStepActions: () => ({ rows: shown.rows, isError: shown.error, retry: () => undefined }),
}));

const { AutomationCardBody } = await import("./AutomationCardBody");

function event(type: string, timestamp: string, payload: Record<string, unknown>): FactoriesWorkOrderEvent {
  return { type, timestamp, event: payload };
}

const EVENTS = [
  event("order.assignees.updated", "2026-01-03T00:00:00Z", {
    run: { id: "run-other" },
    assigned: [{ id: "user-ada" }],
    reason: "Other run.",
  }),
  event("order.assignees.updated", "2026-01-02T00:00:00Z", {
    run: { id: "run-1" },
    assigned: [{ id: "user-ada" }],
    reason: "Changed pkg/payments.",
  }),
  event("order.check.reported", "2026-01-01T00:00:00Z", {
    run: { id: "run-1" },
    check: { name: "Risk score", score: 3, maxScore: 5, format: "fraction", summary: "Low risk overall." },
  }),
  event("order.assignees.updated", "2026-01-04T00:00:00Z", {
    run: { id: "run-1" },
    assigned: [{ id: "user-bea" }],
    reason: "",
  }),
];

function card() {
  const stage: AutomationStage = {
    id: "implement",
    name: "Implement",
    componentName: "Implement",
    status: "passed",
    statusLabel: "Passed",
    duration: "1m",
    checks: [],
    outputs: {
      pullRequests: [],
      artifacts: [{ id: "note", type: "TYPE_MARKDOWN", data: { title: "Notes", filename: "notes.md" } }],
    },
    plumbing: [],
    agentSteps: [],
    steps: [],
    rawLog: "",
  };
  const phase: SplitRunPhase = {
    id: "implement",
    name: "Implement",
    status: "passed",
    duration: "",
    componentName: "Implement",
    artifacts: [],
    stream: [],
    canvasSteps: [],
    runId: "run-1",
  };
  const automation: ConsoleAutomation = { id: "implement", name: "Implement", latest: stage, runs: [stage] };
  return (
    <AutomationCardBody
      automation={automation}
      phase={phase}
      phases={[phase]}
      organizationId="org"
      factoryId="factory"
      orderId="order"
      actionBusy={false}
    />
  );
}

describe("step Actions tab", () => {
  it("stays hidden until the flag is on, then lists this run only", async () => {
    shown.on = false;
    shown.error = false;
    shown.rows = [];
    const view = render(card());

    expect(screen.queryByRole("tab", { name: /Actions/ })).not.toBeInTheDocument();
    expect(screen.queryByText("Changed pkg/payments.")).not.toBeInTheDocument();
    expect(screen.queryByText("Low risk overall.")).not.toBeInTheDocument();
    expect(screen.queryByText("Other run.")).not.toBeInTheDocument();

    shown.on = true;
    shown.rows = stepActionRows(EVENTS, new Set(["run-1"]), (userId) =>
      userId === "user-ada" ? "Ada" : userId === "user-bea" ? "Bea" : undefined,
    );
    view.rerender(card());

    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: "Actions 3" }));

    const titles = screen.getAllByText(/^(Reported|Assigned) /).map((node) => node.textContent);
    expect(titles).toEqual(["Reported Risk score 3/5", "Assigned Ada as owner", "Assigned Bea as owner"]);
    expect(screen.getByText("Low risk overall.")).toBeInTheDocument();
    expect(screen.getByText("Changed pkg/payments.")).toBeInTheDocument();
    expect(screen.queryByText("Other run.")).not.toBeInTheDocument();
    expect(screen.getByText("Assigned Bea as owner").closest("li")?.querySelectorAll("p")).toHaveLength(1);
  });

  it("shows a retry when the action history did not load", async () => {
    shown.on = true;
    shown.rows = [];
    shown.error = true;
    render(card());

    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: "Actions" }));
    expect(screen.getByText("The actions did not load.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });
});
