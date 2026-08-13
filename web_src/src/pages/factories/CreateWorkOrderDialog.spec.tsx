import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { PRIMARY_FACTORY_ID, REFUND_FACTORY } from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { FactoriesLayoutContext } from "./layout/factoriesLayoutContext";

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("./WorkOrderDescriptionEditor", () => ({
  WorkOrderDescriptionEditor: () => <div data-testid="work-order-description-input" />,
}));

vi.mock("./CreateWorkOrderPropertyPills", () => ({
  CreateWorkOrderPropertyPills: () => null,
}));

describe("CreateWorkOrderDialog", () => {
  it("names the dialog New work order instead of the fallback Dialog title", () => {
    render(
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

    expect(screen.getByRole("dialog", { name: "New work order" })).toBeInTheDocument();
    expect(screen.queryByText("Dialog")).not.toBeInTheDocument();
  });
});
