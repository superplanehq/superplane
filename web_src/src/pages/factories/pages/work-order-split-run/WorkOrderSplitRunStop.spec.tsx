import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const { handleStopMock, handleRejectMock, handleArchiveMock, handleBackToDraftMock, enabledExperimentalFeatures } =
  vi.hoisted(() => ({
    handleStopMock: vi.fn(),
    handleRejectMock: vi.fn(),
    handleArchiveMock: vi.fn(),
    handleBackToDraftMock: vi.fn(),
    enabledExperimentalFeatures: new Set<string>(),
  }));

vi.mock("./useSplitRunFooterActions", () => ({
  useSplitRunFooterActions: () => ({
    handleStop: handleStopMock,
    handleReject: handleRejectMock,
    handleArchive: handleArchiveMock,
    handleBackToDraft: handleBackToDraftMock,
    handleStopAutomation: vi.fn(),
    busy: false,
  }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (featureId: string) => enabledExperimentalFeatures.has(featureId),
    enabledExperimentalFeatures: [...enabledExperimentalFeatures],
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({
    data: [{ id: "claude-opus-4-6", name: "claude-opus-4-6" }],
    isLoading: false,
  }),
}));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { FEATURE_FACTORY_DRAFT_START_MODEL } from "@/lib/experimentalFeatures";
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, FAILED_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderPopup(fixture: ComponentProps<typeof WorkOrderSplitRunPopup>["fixture"], onClose?: () => void) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup fixture={fixture} onClose={onClose} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup decision footer", () => {
  beforeEach(() => {
    enabledExperimentalFeatures.clear();
    handleStopMock.mockReset();
    handleRejectMock.mockReset();
    handleArchiveMock.mockReset().mockResolvedValue(true);
    handleBackToDraftMock.mockReset().mockResolvedValue(true);
  });

  it("keeps Reject and Approve off a running task", () => {
    renderPopup(SPLIT_RUN_RUNNING);

    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-review")).not.toBeInTheDocument();
  });

  it("rejects and approves a waiting pull request task from the More menu", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(OPEN_WORK_ORDER));

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "More actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Reject" }));
    expect(handleRejectMock).toHaveBeenCalledTimes(1);
    await user.click(within(note).getByRole("button", { name: "More actions" }));
    await user.click(await screen.findByRole("menuitem", { name: "Approve" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "completed",
      expect.objectContaining({ kind: "waiting", status: "waiting" }),
    );
  });

  it("hides the draft model select when the feature is off", async () => {
    const user = userEvent.setup();
    const onDispatch = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
                onDispatch={onDispatch}
                canDispatch
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).queryByTestId("split-run-draft-model")).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Start" }));
    expect(onDispatch).toHaveBeenCalledWith(undefined);
  });

  it("does not refine a draft from the note", () => {
    const onRefine = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)} onRefine={onRefine} />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(within(screen.getByTestId("split-run-attention-note")).queryByRole("button", { name: "Refine" })).toBeNull();
  });

  it("starts and archives a draft from the note", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_DRAFT_START_MODEL);
    const user = userEvent.setup();
    const onDispatch = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
                onDispatch={onDispatch}
                canDispatch
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).getByTestId("split-run-draft-model")).toHaveTextContent("Auto");
    await user.click(within(note).getByRole("button", { name: "Start" }));
    expect(onDispatch).toHaveBeenCalledTimes(1);
    expect(onDispatch).toHaveBeenCalledWith(undefined);
    expect(screen.getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
    await user.click(within(screen.getByTestId("split-run-attention-note")).getByRole("button", { name: "Archive" }));
    expect(handleArchiveMock).toHaveBeenCalledTimes(1);
  });

  it("closes the popup only after a draft is archived", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER), onClose);

    await user.click(within(screen.getByTestId("split-run-attention-note")).getByRole("button", { name: "Archive" }));

    expect(handleArchiveMock).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does not close a newer popup when a previous archive finishes", async () => {
    let resolveArchive: ((archived: boolean) => void) | undefined;
    handleArchiveMock.mockImplementation(
      () =>
        new Promise<boolean>((resolve) => {
          resolveArchive = resolve;
        }),
    );
    const user = userEvent.setup();
    const firstClose = vi.fn();
    const secondClose = vi.fn();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const popup = (orderId: string, onClose: () => void) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                orderId={orderId}
                fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
                onClose={onClose}
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );
    const view = render(popup("order-a", firstClose));

    await user.click(within(screen.getByTestId("split-run-attention-note")).getByRole("button", { name: "Archive" }));
    view.rerender(popup("order-b", secondClose));
    await act(async () => {
      resolveArchive?.(true);
    });

    expect(firstClose).not.toHaveBeenCalled();
    expect(secondClose).not.toHaveBeenCalled();
  });

  it("keeps the popup open when a draft cannot be archived", async () => {
    handleArchiveMock.mockResolvedValue(false);
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER), onClose);

    await user.click(within(screen.getByTestId("split-run-attention-note")).getByRole("button", { name: "Archive" }));

    expect(handleArchiveMock).toHaveBeenCalledTimes(1);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("starts a draft with the listed model", async () => {
    enabledExperimentalFeatures.add(FEATURE_FACTORY_DRAFT_START_MODEL);
    const user = userEvent.setup();
    const onDispatch = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
                onDispatch={onDispatch}
                canDispatch
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const note = screen.getByTestId("split-run-attention-note");
    await user.click(within(note).getByRole("combobox", { name: "Model" }));
    await user.click(await screen.findByRole("option", { name: "claude-opus-4-6" }));
    await user.click(within(note).getByRole("button", { name: "Start" }));
    expect(onDispatch).toHaveBeenCalledWith("claude-opus-4-6");
  });

  it("opens Description after To Backlog", async () => {
    const user = userEvent.setup();
    renderPopup(
      splitRunFixtureForWorkOrder({
        id: "wo-stopped",
        title: "Stopped job",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_CANCELLED",
              },
            ],
          },
        ],
      }),
    );

    expect(screen.getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
    await user.click(
      within(screen.getByTestId("split-run-attention-note")).getByRole("button", { name: "To Backlog" }),
    );
    expect(handleBackToDraftMock).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("tab", { name: "Description" })).toHaveAttribute("data-state", "active");
  });

  it("keeps Automations open when To Backlog does not succeed", async () => {
    handleBackToDraftMock.mockResolvedValueOnce(false);
    const user = userEvent.setup();
    renderPopup(
      splitRunFixtureForWorkOrder({
        id: "wo-stopped",
        title: "Stopped job",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl",
                step: "Implement",
                stepIndex: 0,
                state: "STATE_FINISHED",
                result: "RESULT_CANCELLED",
              },
            ],
          },
        ],
      }),
    );

    expect(screen.getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
    await user.click(
      within(screen.getByTestId("split-run-attention-note")).getByRole("button", { name: "To Backlog" }),
    );
    expect(handleBackToDraftMock).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
  });

  it("reruns a failed open task from the note", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(FAILED_WORK_ORDER));

    const note = screen.getByTestId("split-run-attention-note");
    expect(within(note).queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Rerun" }));
    expect(handleStopMock).toHaveBeenCalledWith("rerun-step", expect.objectContaining({ kind: "failed" }));
  });

  it("reopens a closed task from the note", async () => {
    const user = userEvent.setup();
    renderPopup(splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER));

    const note = screen.getByTestId("split-run-attention-note");
    expect(screen.queryByTestId("split-run-header-actions")).not.toBeInTheDocument();
    expect(within(note).queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Reopen" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "reopen",
      expect.objectContaining({ kind: "failed", status: "failed" }),
    );
  });
});
