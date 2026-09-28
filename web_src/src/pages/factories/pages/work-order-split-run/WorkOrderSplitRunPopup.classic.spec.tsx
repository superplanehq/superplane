import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { LINE_BOARD_DONE_RECEIPTS_ORDER } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { SPLIT_RUN_SUPER503_RUNNING } from "./splitRunSuper503RunningFixture";

// The classic description tab embeds Monaco, which cannot load in Happy DOM.
vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

// Without the Task Console experimental feature, the popup keeps the classic tabs.
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => false,
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

function renderPopup(props: ComponentProps<typeof WorkOrderSplitRunPopup>) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup {...props} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup without the Task Console feature", () => {
  it("keeps the classic Task and Automations tabs instead of the console", () => {
    renderPopup({ fixture: SPLIT_RUN_SUPER503_RUNNING });

    expect(screen.getByRole("tab", { name: "Task" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Automations" })).toBeInTheDocument();
    expect(screen.queryByTestId("redesign-console-variant")).not.toBeInTheDocument();
  });

  it("hides the console-only closure phase from the classic log", async () => {
    const user = userEvent.setup();
    const fixture = splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER);
    expect(fixture.phases.some((phase) => phase.id === "done-closure")).toBe(true);

    renderPopup({ fixture });
    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(document.querySelector("[data-testid^='split-run-phase-']")).not.toBeNull();
    expect(screen.queryByTestId("split-run-phase-done-closure")).not.toBeInTheDocument();
  });
});
