import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PRIMARY_FACTORY_ID, REFUND_FACTORY } from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { FactoriesLayoutContext } from "./layout/factoriesLayoutContext";

const { canAct } = vi.hoisted(() => ({
  canAct: vi.fn((resource: string, action: string) => resource === "work_orders" && action === "create"),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct,
    isLoading: false,
  }),
}));

vi.mock("./WorkOrderDescriptionEditor", () => ({
  WorkOrderDescriptionEditor: () => <div data-testid="work-order-description-input" />,
}));

vi.mock("./CreateWorkOrderPropertyPills", () => ({
  CreateWorkOrderPropertyPills: () => null,
}));

function renderDialog() {
  return render(
    <FactoriesLayoutContext.Provider
      value={{
        organizationId: "org-1",
        factoryId: PRIMARY_FACTORY_ID,
        factory: REFUND_FACTORY,
        factories: [REFUND_FACTORY],
        openCreateWorkOrder: vi.fn(),
      }}
    >
      <CreateWorkOrderDialog open onClose={vi.fn()} onCreated={vi.fn()} />
    </FactoriesLayoutContext.Provider>,
  );
}

describe("CreateWorkOrderDialog", () => {
  beforeEach(() => {
    canAct.mockImplementation((resource: string, action: string) => resource === "work_orders" && action === "create");
  });

  it("names the dialog New work order instead of the fallback Dialog title", () => {
    renderDialog();

    expect(screen.getByRole("dialog", { name: "New work order" })).toBeInTheDocument();
    expect(screen.queryByText("Dialog")).not.toBeInTheDocument();
  });

  it("disables Send to line when the user cannot dispatch work orders", () => {
    renderDialog();

    expect(screen.getByTestId("work-order-create-send-to-line").closest(".pointer-events-none")).toBeInTheDocument();
  });
});
