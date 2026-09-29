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
const useCreateWorkOrderSpy = vi.fn();
vi.mock("@/hooks/useFactoryData", async () => {
  const actual = await vi.importActual("@/hooks/useFactoryData");
  return {
    ...actual,
    useCreateWorkOrder: (organizationId: string, factoryId: string) => {
      useCreateWorkOrderSpy(organizationId, factoryId);
      return {
        mutateAsync: createWorkOrderMutate,
        mutate: vi.fn(),
        isPending: false,
      };
    },
  };
});

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
  showInfoToast: vi.fn(),
}));

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { showErrorToast } from "@/lib/toast";
import { workOrderFileRef } from "@/lib/workOrderFiles";

import {
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../__fixtures__/lineMetricsBoardOrders";
import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

async function flushPromises() {
  await act(async () => {
    await Promise.resolve();
  });
}

function renderPopup({ canCreateWorkOrder, description }: { canCreateWorkOrder?: boolean; description?: string } = {}) {
  const order =
    description === undefined ? BOARD_IMPLEMENT_NOTIFY_ORDER : { ...BOARD_IMPLEMENT_NOTIFY_ORDER, description };
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup
              organizationId={FACTORIES_ORGANIZATION_ID}
              factoryId={PRIMARY_FACTORY_ID}
              factoryKey={PRIMARY_FACTORY_KEY}
              orderNumber={BOARD_IMPLEMENT_NOTIFY_ORDER.number}
              fixture={splitRunFixtureForWorkOrder(order)}
              canCreateWorkOrder={canCreateWorkOrder}
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
    useCreateWorkOrderSpy.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a duplicate button in the header", () => {
    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).toBeInTheDocument();
  });

  it("scopes task creation to the current organization and factory", () => {
    renderPopup();

    expect(useCreateWorkOrderSpy).toHaveBeenCalledWith(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_ID);
  });

  it("creates a new task with the current title and description on click", async () => {
    createWorkOrderMutate.mockResolvedValue({
      id: "new-order-id",
      number: "100",
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

  it("strips embedded file references from the duplicated description", async () => {
    const fileId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
    createWorkOrderMutate.mockResolvedValue({
      id: "new-order-id",
      number: "100",
      title: BOARD_IMPLEMENT_NOTIFY_ORDER.title,
      description: "Original description",
    });

    renderPopup({ description: `Steps to reproduce.\n\n![screenshot](${workOrderFileRef(fileId)})` });

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.click(button);

    await flushPromises();

    const call = createWorkOrderMutate.mock.calls.at(-1)?.[0] as { description: string };
    expect(call.description).not.toContain(workOrderFileRef(fileId));
    expect(call.description).toContain("Steps to reproduce.");
  });

  it("navigates to the new task after successful creation", async () => {
    createWorkOrderMutate.mockResolvedValue({
      id: "new-order-id",
      number: "100",
      title: BOARD_IMPLEMENT_NOTIFY_ORDER.title,
      description: "Original description",
    });

    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.click(button);

    await flushPromises();

    const expectedPath = workOrderDetailPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "100");
    expect(mockNavigate).toHaveBeenCalledWith(expectedPath);
  });

  it("shows an error toast and stays on the task when creation fails", async () => {
    createWorkOrderMutate.mockRejectedValue(new Error("Failed to create task"));

    renderPopup();

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    fireEvent.click(button);

    await flushPromises();

    expect(showErrorToast).toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("disables the duplicate button when the user cannot create tasks", async () => {
    renderPopup({ canCreateWorkOrder: false });

    const button = screen.getByTestId("popup-work-order-duplicate-button");
    expect(button).toBeDisabled();

    fireEvent.click(button);
    await flushPromises();

    expect(createWorkOrderMutate).not.toHaveBeenCalled();
  });
});
