import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as FactoryData from "@/hooks/useFactoryData";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const createMutate = vi.hoisted(() => vi.fn());
const successToast = vi.hoisted(() => vi.fn());
const permissions = vi.hoisted(() => ({ canCreate: true }));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: successToast,
  showErrorToast: vi.fn(),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) =>
      permissions.canCreate && resource === "work_orders" && action === "create",
    isLoading: false,
    permissions: [],
    currentUserId: "user-1",
    browserNotificationPreferences: { enabled: false, showWhileViewing: true },
  }),
}));

vi.mock("@/hooks/useFactoryData", () => {
  const actual = unmockedSrc<typeof FactoryData>("hooks/useFactoryData");
  return {
    ...actual,
    useFactory: () => ({
      data: { id: "factory-1", planning: { enabled: false, clarity: false, confidence: false } },
      isPending: false,
    }),
    useCreateWorkOrder: () => ({ mutateAsync: createMutate, isPending: false }),
  };
});

const SAVED_TITLE = "Retry refunds";
const SAVED_DESCRIPTION = "Refunds fail.\n\n![Checkout](sp-file://file-1)\n\nMore context.";

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{`${location.pathname}${location.search}`}</div>;
}

function renderPopup(onDispatch = vi.fn()) {
  const order = {
    ...OPEN_WORK_ORDER,
    title: SAVED_TITLE,
    description: SAVED_DESCRIPTION,
  };
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/org-1/workspaces/acme/task/101?lineId=line-1"]}>
        <ThemeProvider>
          <TooltipProvider>
            <LocationProbe />
            <WorkOrderSplitRunPopup
              organizationId="org-1"
              factoryId="factory-1"
              factoryKey="ACME"
              orderId="work-order-1"
              orderNumber="101"
              lineId="line-1"
              fixture={splitRunFixtureForWorkOrder(order)}
              onDispatch={onDispatch}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return onDispatch;
}

describe("WorkOrderSplitRunPopup duplicate", () => {
  beforeEach(() => {
    permissions.canCreate = true;
    createMutate.mockReset();
    createMutate.mockResolvedValue({ id: "wo-new", number: "202", title: SAVED_TITLE });
    successToast.mockReset();
  });

  it("creates a draft from the saved title and description and does not dispatch", async () => {
    const user = userEvent.setup();
    const onDispatch = renderPopup();

    await user.click(screen.getByRole("button", { name: "Duplicate" }));

    await waitFor(() => {
      expect(createMutate).toHaveBeenCalledWith({
        title: SAVED_TITLE,
        description: "Refunds fail.\n\nMore context.",
      });
    });
    expect(createMutate.mock.calls[0]?.[0]).not.toHaveProperty("assigneeIds");
    expect(onDispatch).not.toHaveBeenCalled();
    expect(successToast).toHaveBeenCalledWith("Task duplicated.");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/acme/task/202?lineId=line-1");
  });

  it("hides Duplicate when create is not allowed", () => {
    permissions.canCreate = false;
    renderPopup();

    expect(screen.queryByRole("button", { name: "Duplicate" })).not.toBeInTheDocument();
  });
});
