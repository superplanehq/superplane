import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrderEvent } from "@/api-client";
import type { AutomationStage, ConsoleAutomation } from "./automationsViewModel";
import { stepActionRows } from "./consoleStepActions";
import type { StepActionRow } from "./consoleStepActions";
import type { SplitRunPhase } from "../splitRunMocks";

const shown: { on: boolean; rows: StepActionRow[]; error: boolean } = { on: false, rows: [], error: false };
const retryActions = vi.fn();

vi.mock("@/hooks/useFactoryGitHubHost", () => ({
  useFactoryGitHubHost: () => "",
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => shown.on && id === "factory_step_actions",
    enabledExperimentalFeatures: [],
    isLoading: false,
    organizationReady: true,
  }),
}));

vi.mock("./useStepActions", () => ({
  useStepActions: () => ({ rows: shown.rows, isError: shown.error, retry: retryActions }),
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

function card(runCount = 1, startedAt?: string) {
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
  const runs = Array.from({ length: runCount }, (_, index) => ({
    ...stage,
    id: index === 0 ? "implement" : `implement-${index + 1}`,
    startedAt: startedAt ?? `2026-01-01T0${index}:00:00Z`,
  }));
  const phases = runs.map((run, index) => ({ ...phase, id: run.id, runId: `run-${index + 1}` }));
  const automation: ConsoleAutomation = { id: "implement", name: "Implement", latest: runs.at(-1)!, runs };
  return (
    <AutomationCardBody
      automation={automation}
      phase={phase}
      phases={phases}
      organizationId="org"
      factoryId="factory"
      orderId="order"
      actionBusy={false}
    />
  );
}

describe("step Actions tab", () => {
  it("uses a valid action time when the run time is invalid and omits unavailable time", async () => {
    shown.on = true;
    shown.error = false;
    shown.rows = stepActionRows(EVENTS, new Set(["run-1"]), () => "Ada");
    const view = render(card(1, "invalid"));
    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: "Actions 3" }));
    const trigger = screen.getByRole("button", { name: /Implement.*3 actions/ });
    expect(trigger.querySelector("time")).toHaveAttribute("datetime", "2026-01-01T00:00:00Z");
    shown.rows = stepActionRows(
      EVENTS.map((report) => ({ ...report, timestamp: "invalid" })),
      new Set(["run-1"]),
      () => "Ada",
    );
    view.rerender(card(1, "invalid"));
    expect(trigger.querySelector("time")).toBeNull();
  });

  it("discloses same-day runs independently and preserves chosen state when history updates", async () => {
    shown.on = true;
    shown.error = false;
    const reports = [
      event("order.check.reported", "2026-01-01T01:02:00Z", {
        run: { id: "run-2" },
        check: { name: "Risk score", score: 4, maxScore: 5, summary: "Second report." },
      }),
      event("order.check.reported", "2026-01-01T01:01:00Z", {
        run: { id: "run-2" },
        check: { name: "Risk score", score: 2, maxScore: 5, summary: "First report." },
      }),
      ...EVENTS,
    ];
    shown.rows = stepActionRows(reports, new Set(["run-1", "run-2"]), () => "Ada");
    const view = render(card(3));
    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: "Actions 5" }));
    const older = screen.getByRole("button", { name: /Implement.*Run 1.*3 actions/ });
    const newer = screen.getByRole("button", { name: /Implement.*Run 2.*2 actions/ });
    expect(older).toHaveAttribute("aria-expanded", "false");
    expect(newer).toHaveAttribute("aria-expanded", "true");
    expect(screen.queryByText("Changed pkg/payments.")).not.toBeInTheDocument();
    expect(screen.getAllByText(/^Reported Risk score/).map((node) => node.textContent)).toEqual([
      "Reported Risk score 2/5",
      "Reported Risk score 4/5",
    ]);
    await user.click(older);
    expect(screen.getByText("Changed pkg/payments.")).toBeInTheDocument();
    expect(newer).toHaveAttribute("aria-expanded", "true");
    await user.click(newer);
    shown.rows = stepActionRows(
      [
        event("order.assignees.updated", "2026-01-01T01:03:00Z", {
          run: { id: "run-2" },
          assigned: [{ id: "user-ada" }],
          reason: "Updated owner.",
        }),
        ...reports,
      ],
      new Set(["run-1", "run-2"]),
      () => "Ada",
    );
    shown.error = true;
    retryActions.mockClear();
    view.rerender(card(3));
    expect(older).toHaveAttribute("aria-expanded", "true");
    expect(newer).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(retryActions).toHaveBeenCalledTimes(1);
    newer.focus();
    await user.keyboard("{Enter}");
    expect(screen.getByText("Updated owner.")).toBeInTheDocument();
    expect(screen.queryByText("Other run.")).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: /Artifacts/ }));
    await user.click(screen.getByRole("tab", { name: "Actions 6" }));
    expect(older).toHaveAttribute("aria-expanded", "true");
  });

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
    await user.click(screen.getByRole("button", { name: /Implement.*3 actions/ }));
    expect(screen.queryByText("Changed pkg/payments.")).not.toBeInTheDocument();
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
