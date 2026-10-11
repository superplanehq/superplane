import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { SPLIT_RUN_RUNNING } from "../pages/work-order-split-run/splitRunMocks";
import type { SplitRunFixture } from "../pages/work-order-split-run/splitRunMocks";
import { MobileTaskActivity } from "./MobileTaskActivity";

vi.mock("../pages/work-order-split-run/useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas: () => ({
    enabled: false,
    isError: false,
    isLoading: false,
    canvas: undefined,
    stream: [],
  }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useWorkOrder: () => ({ data: undefined }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
}));

vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: () => ({ data: [] }),
}));

function transcriptFixture(): SplitRunFixture {
  const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
  return {
    ...SPLIT_RUN_RUNNING,
    lineStatus: "running",
    footerTone: "running",
    currentPhaseId: "implement",
    footer: {
      ...SPLIT_RUN_RUNNING.footer,
      kind: "running",
      run: { appId: "app-refund-implementer", runId: "run-1" },
    },
    phases: [
      {
        ...(implement ?? SPLIT_RUN_RUNNING.phases[0]),
        id: "implement",
        name: "Implement",
        status: "running",
        componentName: "Implementation",
        appId: "app-refund-implementer",
        runId: "run-1",
        startedAt: new Date("2026-10-08T20:22:00Z").toISOString(),
        model: "deepseek/deepseek-v4-flash",
        totalTokens: "24600",
        stream: [
          {
            id: "step-clone",
            at: "20:22:01",
            note: true,
            componentType: "bash",
            componentName: "Clone Repo",
            status: "passed",
            detail: "Cloning into 'repo'...",
          },
          {
            id: "step-implement",
            at: "20:22:09",
            note: true,
            componentType: "prompt",
            componentName: "Implementation",
            status: "passed",
            detail: "You are implementing a SuperPlane task. The repository is cloned into repo.",
          },
          {
            id: "tool-read",
            at: "20:22:11",
            note: true,
            noteParentId: "step-implement",
            componentType: "read",
            componentName: "src/scenarios/formatTax.ts",
            status: "passed",
          },
          {
            id: "step-commit",
            at: "20:23:40",
            note: true,
            componentType: "bash",
            componentName: "Commit and Push",
            status: "passed",
            detail: "git push origin fix/format-tax",
          },
          {
            id: "step-pr",
            at: "20:24:10",
            note: true,
            componentType: "prompt",
            componentName: "Generate PR title and description",
            status: "running",
            detail: "Summarize the branch into a pull request title and description.",
          },
          {
            id: "note-pr",
            at: "20:24:31",
            note: true,
            noteParentId: "step-pr",
            componentType: "note",
            componentName: "Done. Branch fix/format-tax pushed with two changes.",
            status: "passed",
          },
        ],
      },
    ],
  };
}

function renderActivity(fixture: SplitRunFixture) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <MobileTaskActivity
              organizationId="org-1"
              factoryId="factory-1"
              orderId="wo-1"
              fixture={fixture}
              factoryKey="workspace"
              orderNumber="103"
              canStopRun
              onStopRun={vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("MobileTaskActivity console cards", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("shows the Implement column and card header like the desktop console", () => {
    renderActivity(transcriptFixture());

    const column = screen.getByTestId("mobile-task-column-implement");
    expect(screen.getByTestId("mobile-task-column-marker-implement")).toHaveAttribute("data-status", "running");
    expect(within(column).getByText("1 agent run")).toBeInTheDocument();
    expect(within(column).getAllByText("Implementation").length).toBeGreaterThanOrEqual(1);
  });

  it("lists collapsible agent steps with their prompts", async () => {
    const user = userEvent.setup();
    renderActivity(transcriptFixture());

    for (const title of ["Clone Repo", "Commit and Push", "Generate PR title and description"]) {
      expect(screen.getByText(title)).toBeInTheDocument();
    }
    expect(screen.getAllByText("Implementation").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText(/You are implementing a SuperPlane task/)).toBeInTheDocument();

    const cloneToggle = screen.getByRole("button", { name: /Clone Repo/ });
    expect(cloneToggle).toHaveAttribute("aria-expanded", "false");
    await user.click(cloneToggle);
    expect(cloneToggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText(/Cloning into/)).toBeInTheDocument();
  });

  it("shows the run footer with spend, model, and Stop", () => {
    const onStopRun = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <MobileTaskActivity
                organizationId="org-1"
                factoryId="factory-1"
                orderId="wo-1"
                fixture={transcriptFixture()}
                factoryKey="workspace"
                orderNumber="103"
                canStopRun
                onStopRun={onStopRun}
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const footer = screen.getByTestId("redesign-console-run-footer-implement");
    expect(footer).toHaveTextContent("24.6k");
    expect(footer).toHaveTextContent("deepseek-v4-flash");
    expect(within(footer).getByRole("button", { name: "Stop" })).toBeInTheDocument();
  });
});
