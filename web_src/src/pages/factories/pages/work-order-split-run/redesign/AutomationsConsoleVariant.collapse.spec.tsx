import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { TWO_HOURS_AGO } from "../../../__fixtures__/factoryPageIds";
import { LiveHeaderSpendProvider } from "../liveHeaderSpendContext";
import { buildSplitRunFooter } from "../splitRunFooter";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";

type ConsoleProps = Parameters<typeof AutomationsConsoleVariant>[0];

function renderConsole(fixture: typeof SPLIT_RUN_RUNNING, extra: Partial<ConsoleProps> = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const ui = (next: typeof fixture) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <LiveHeaderSpendProvider>
              <AutomationsConsoleVariant fixture={next} source={next.source} {...extra} />
            </LiveHeaderSpendProvider>
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(ui(fixture));
  return { ...view, rerenderConsole: (next: typeof fixture) => view.rerender(ui(next)) };
}

function cardState(name: string) {
  return screen.getByRole("button", { name: `Toggle ${name} details` }).closest("[data-state]");
}

describe("AutomationsConsoleVariant card collapse", () => {
  it("keeps a running card open and collapses finished cards", () => {
    renderConsole(SPLIT_RUN_RUNNING);

    expect(cardState("Implementation")).toHaveAttribute("data-state", "open");
    expect(screen.queryByRole("button", { name: "Toggle Ingest details" })).not.toBeInTheDocument();
  });

  it("collapses Backlog when it finishes while Implement is still running", () => {
    const overlapping = {
      ...SPLIT_RUN_RUNNING,
      phases: [
        ...SPLIT_RUN_RUNNING.phases,
        {
          id: "analysis-1",
          name: "Analysis",
          status: "running" as const,
          duration: "30s",
          startedAt: TWO_HOURS_AGO,
          componentName: "Backlog",
          appId: "app-refund-backlog-analyzer",
          artifacts: [],
          stream: [],
          canvasSteps: [],
        },
      ],
    };
    const { rerenderConsole } = renderConsole(overlapping);

    expect(cardState("Backlog")).toHaveAttribute("data-state", "open");
    expect(cardState("Implementation")).toHaveAttribute("data-state", "open");

    rerenderConsole({
      ...overlapping,
      phases: overlapping.phases.map((phase) =>
        phase.id === "analysis-1" ? { ...phase, status: "passed" as const } : phase,
      ),
    });

    expect(cardState("Backlog")).toHaveAttribute("data-state", "closed");
    expect(cardState("Implementation")).toHaveAttribute("data-state", "open");
  });

  it("keeps a closed draft card closed after Backlog finishes", () => {
    const draftBacklog = {
      ...SPLIT_RUN_RUNNING,
      lineStatus: "pending" as const,
      footerTone: "draft" as const,
      footer: buildSplitRunFooter({ kind: "draft" }),
      phases: [
        ...SPLIT_RUN_RUNNING.phases.filter((phase) => phase.id !== "implement"),
        {
          id: "analysis-1",
          name: "Analysis",
          status: "running" as const,
          duration: "30s",
          startedAt: TWO_HOURS_AGO,
          componentName: "Backlog",
          appId: "app-refund-backlog-analyzer",
          artifacts: [],
          stream: [],
          canvasSteps: [],
        },
      ],
    };
    const { rerenderConsole } = renderConsole(draftBacklog);

    fireEvent.click(screen.getByRole("button", { name: "Toggle Backlog details" }));

    rerenderConsole({
      ...draftBacklog,
      phases: draftBacklog.phases.map((phase) =>
        phase.id === "analysis-1" ? { ...phase, status: "passed" as const } : phase,
      ),
    });

    expect(cardState("Backlog")).toHaveAttribute("data-state", "closed");
  });

  it("collapses the last card after every automation finishes", () => {
    const { rerenderConsole } = renderConsole(SPLIT_RUN_RUNNING);
    expect(cardState("Implementation")).toHaveAttribute("data-state", "open");

    rerenderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "passed",
      footerTone: "done",
      phases: SPLIT_RUN_RUNNING.phases.map((phase) => ({ ...phase, status: "passed" })),
    });

    expect(cardState("Implementation")).toHaveAttribute("data-state", "closed");
    expect(screen.queryByRole("button", { name: "Toggle Ingest details" })).not.toBeInTheDocument();
  });

  it("sums every Implement run on the card header", () => {
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      lineStatus: "passed",
      footerTone: "done",
      phases: SPLIT_RUN_RUNNING.phases.flatMap((phase) =>
        phase.id === "implement"
          ? [
              { ...phase, id: "implement-old", status: "passed" as const, duration: "22m 38s", artifacts: [] },
              { ...phase, status: "passed" as const, duration: "9m 22s" },
            ]
          : [phase],
      ),
    });

    expect(screen.getByTestId("redesign-console-card-header-implementation-implementation")).toHaveTextContent("32m");
  });
});
