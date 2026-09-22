import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import type * as FactoryData from "@/hooks/useFactoryData";
import { unmockedSrc } from "@/test/unmockedModule";

const { handleStopMock, handleRejectMock, handleArchiveMock, factoryPlanning } = vi.hoisted(() => ({
  handleStopMock: vi.fn(),
  handleRejectMock: vi.fn(),
  handleArchiveMock: vi.fn(),
  factoryPlanning: { current: { enabled: true, clarity: true, confidence: true } },
}));

vi.mock("./useSplitRunFooterActions", () => ({
  useSplitRunFooterActions: () => ({
    handleStop: handleStopMock,
    handleReject: handleRejectMock,
    handleArchive: handleArchiveMock,
    handleStopAutomation: vi.fn(),
    busy: false,
  }),
}));

vi.mock("@/hooks/useFactoryData", () => {
  const actual = unmockedSrc<typeof FactoryData>("hooks/useFactoryData");
  return {
    ...actual,
    useFactory: () => ({
      data: { id: "factory-1", planning: factoryPlanning.current },
      isPending: false,
    }),
  };
});

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
import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER, FAILED_WORK_ORDER, OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
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
    window.localStorage.clear();
    factoryPlanning.current = { enabled: true, clarity: true, confidence: true };
    handleStopMock.mockReset();
    handleRejectMock.mockReset();
    handleArchiveMock.mockReset().mockResolvedValue(true);
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
    expect(within(note).queryByRole("button", { name: "More actions" })).not.toBeInTheDocument();
    const moreActions = screen.getByRole("button", { name: "More actions" });
    expect(moreActions.closest("header")).not.toBeNull();
    await user.click(moreActions);
    await user.click(await screen.findByRole("menuitem", { name: "Reject" }));
    expect(handleRejectMock).toHaveBeenCalledTimes(1);
    await user.click(moreActions);
    await user.click(await screen.findByRole("menuitem", { name: "Approve" }));
    expect(handleStopMock).toHaveBeenCalledWith(
      "completed",
      expect.objectContaining({ kind: "waiting", status: "waiting" }),
    );
  });

  it("hides the draft model chevron on Start when Planning is off", async () => {
    factoryPlanning.current = { enabled: false, clarity: true, confidence: true };
    const user = userEvent.setup();
    const onDispatch = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                factoryId="factory-1"
                orderId={DRAFT_WORK_ORDER.id}
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
    expect(within(note).queryByRole("button", { name: /^Model/ })).not.toBeInTheDocument();
    await user.click(within(note).getByRole("button", { name: "Start" }));
    expect(onDispatch).toHaveBeenCalledWith(undefined);
  });

  it("does not refine a draft from the note", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)} />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.queryByRole("button", { name: "Refine" })).toBeNull();
  });

  it("starts and archives a draft from the note", async () => {
    const user = userEvent.setup();
    const onDispatch = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                factoryId="factory-1"
                orderId={DRAFT_WORK_ORDER.id}
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
    expect(within(note).getByRole("button", { name: "Model: Auto" })).toBeInTheDocument();
    expect(within(note).getByTestId("split-run-draft-model")).not.toHaveTextContent("Auto");
    await user.click(within(note).getByRole("button", { name: "Start" }));
    expect(onDispatch).toHaveBeenCalledTimes(1);
    expect(onDispatch).toHaveBeenCalledWith(undefined);
    expect(screen.getByRole("tab", { name: "Automations" })).toHaveAttribute("data-state", "active");
    await user.click(screen.getByTestId("popup-work-order-archive-button"));
    expect(handleArchiveMock).toHaveBeenCalledTimes(1);
  });

  it("closes the popup only after a draft is archived", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderPopup(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER), onClose);

    await user.click(screen.getByTestId("popup-work-order-archive-button"));

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

    await user.click(screen.getByTestId("popup-work-order-archive-button"));
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

    await user.click(screen.getByTestId("popup-work-order-archive-button"));

    expect(handleArchiveMock).toHaveBeenCalledTimes(1);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("starts a draft with the listed model", async () => {
    const user = userEvent.setup();
    const onDispatch = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <WorkOrderSplitRunPopup
                fixture={splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0])}
                onDispatch={onDispatch}
                canDispatch
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const strip = screen.getByTestId("split-run-intent-status-card");
    const settings = within(strip).getByTestId("split-run-intent-settings");
    const model = within(settings).getByRole("button", { name: "Model: Auto" });
    expect(model).toHaveTextContent("Auto");
    await user.click(model);
    await user.click(await screen.findByRole("menuitemradio", { name: "claude-opus-4-6" }));
    expect(within(settings).getByRole("button", { name: "Model: claude-opus-4-6" })).toBeInTheDocument();
    const actions = within(strip).getByTestId("split-run-draft-action-group");
    expect(within(actions).queryByTestId("split-run-draft-model")).not.toBeInTheDocument();
    await user.click(within(actions).getByRole("button", { name: "Start" }));
    expect(onDispatch).toHaveBeenCalledWith("claude-opus-4-6");
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
