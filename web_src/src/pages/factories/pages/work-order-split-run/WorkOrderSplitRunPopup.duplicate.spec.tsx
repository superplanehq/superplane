import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: () => undefined,
  useCanvasWebsocket: () => undefined,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: undefined, isError: false, isLoading: false }),
  useDescribeRun: () => ({ data: undefined, isError: false, isLoading: false }),
}));

const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
  const actual = await vi.importActual("react-router");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const createWorkOrderMutate = vi.fn();
vi.mock("@/hooks/useFactoryData", async () => {
  const actual = await vi.importActual("@/hooks/useFactoryData");
  return {
    ...actual,
    useCreateWorkOrder: () => ({
      mutateAsync: createWorkOrderMutate,
      mutate: vi.fn(),
      isPending: false,
    }),
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

async function flushPromises() {
  await act(async () => {
    await Promise.resolve();
  });
}

function renderPopup() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup
              organizationId={FACTORIES_ORGANIZATION_ID}
              factoryKey={PRIMARY_FACTORY_KEY}
              orderNumber={BOARD_IMPLEMENT_NOTIFY_ORDER.number}
              fixture={splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER)}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup duplicate button", () => {
  beforeEach(() => {
    mockNavigate.mockReset();
    createWorkOrderMutate.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a duplicate icon button in the header", () => {
    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).toBeInTheDocument();
  });

  it("creates a new task with same title and body on click", async () => {
    const newTaskNumber = "100";
    createWorkOrderMutate.mockResolvedValue({
      id: "new-order-id",
      number: newTaskNumber,
      title: BOARD_IMPLEMENT_NOTIFY_ORDER.title,
      description: "Original description",
    });

    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.click(button);

    await flushPromises();

    expect(createWorkOrderMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        title: BOARD_IMPLEMENT_NOTIFY_ORDER.title,
        description: expect.any(String),
      }),
    );
  });

  it("navigates to the new task after successful creation", async () => {
    const newTaskNumber = "100";
    createWorkOrderMutate.mockResolvedValue({
      id: "new-order-id",
      number: newTaskNumber,
      title: BOARD_IMPLEMENT_NOTIFY_ORDER.title,
      description: "Original description",
    });

    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.click(button);

    await flushPromises();

    const expectedPath = workOrderDetailPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, newTaskNumber);
    expect(mockNavigate).toHaveBeenCalledWith(expectedPath);
  });

  it("shows error toast on creation failure", async () => {
    const { showErrorToast } = await import("@/lib/toast");
    createWorkOrderMutate.mockRejectedValue(new Error("Failed to create task"));

    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.click(button);

    await flushPromises();

    expect(showErrorToast).toHaveBeenCalled();
  });
});
