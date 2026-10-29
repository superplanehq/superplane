import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrderEvent } from "@/api-client";
import type { ConsoleAutomation } from "./automationsViewModel";
import type { SplitRunPhase } from "../splitRunMocks";
import { useStepActions } from "./useStepActions";

const { fetchNextPage } = vi.hoisted(() => ({ fetchNextPage: vi.fn() }));
const gate: { on: boolean; events: FactoriesWorkOrderEvent[]; hasNextPage: boolean; isError: boolean } = {
  on: false,
  events: [],
  hasNextPage: false,
  isError: false,
};

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => gate.on && id === "factory_step_actions",
    enabledExperimentalFeatures: [],
    isLoading: false,
    organizationReady: true,
  }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useWorkOrderEvents: () => ({
    data: { pages: [{ events: gate.events }] },
    hasNextPage: gate.hasNextPage,
    isFetchingNextPage: false,
    isError: gate.isError,
    fetchNextPage,
    refetch: () => undefined,
  }),
}));

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (userId?: string) => {
      if (userId === "user-ada") {
        return { id: userId, name: "Ada", initials: "A" };
      }
      return null;
    },
    isLoading: false,
  }),
}));

function Probe({ phases }: { phases: SplitRunPhase[] }) {
  const automation = {
    id: "implement",
    name: "Implement",
    latest: { id: "implement" },
    runs: [{ id: "implement" }],
  } as ConsoleAutomation;
  const actions = useStepActions("org", "factory", "order", automation, phases);
  return <pre>{actions.rows.map((row) => `${row.title}|${row.detail ?? ""}`).join("\n")}</pre>;
}

const phase = { id: "implement", runId: "run-1" } as SplitRunPhase;

describe("useStepActions", () => {
  it("returns nothing while the flag is off and hides another run when the flag is on", () => {
    gate.on = false;
    gate.hasNextPage = false;
    gate.events = [
      {
        type: "order.assignees.updated",
        timestamp: "2026-01-02T00:00:00Z",
        event: { run: { id: "run-other" }, assigned: [{ id: "user-ada" }], reason: "Other run." },
      },
      {
        type: "order.check.reported",
        timestamp: "2026-01-01T00:00:00Z",
        event: {
          run: { id: "run-1" },
          check: { name: "Risk score", score: 3, maxScore: 5, format: "fraction", summary: "Low risk overall." },
        },
      },
    ];
    const view = render(<Probe phases={[phase]} />);
    expect(view.container.querySelector("pre")?.textContent).toBe("");

    gate.on = true;
    view.rerender(<Probe phases={[phase]} />);
    expect(screen.getByText("Reported Risk score 3/5|Low risk overall.")).toBeInTheDocument();
    expect(screen.queryByText(/Other run/)).not.toBeInTheDocument();
  });

  it("loads the next event page while more history exists", () => {
    gate.on = true;
    gate.hasNextPage = true;
    gate.events = [];
    fetchNextPage.mockClear();
    render(<Probe phases={[phase]} />);
    expect(fetchNextPage).toHaveBeenCalled();
  });
});
