import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  EMPTY_FACTORY,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
} from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { FactoriesLayoutContext } from "./layout/factoriesLayoutContext";

const { canAct, permissions, createMutate, dispatchMutate, meUser } = vi.hoisted(() => ({
  canAct: vi.fn((resource: string, action: string) => resource === "work_orders" && action === "create"),
  permissions: { isLoading: false },
  createMutate: vi.fn(),
  dispatchMutate: vi.fn(),
  meUser: { current: null as { id: string; name: string } | null },
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: createMutate, isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: dispatchMutate, isPending: false }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: meUser.current }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct,
    isLoading: permissions.isLoading,
  }),
}));

vi.mock("./WorkOrderDescriptionEditor", () => ({
  WorkOrderDescriptionEditor: () => <div data-testid="work-order-description-input" />,
}));

function renderDialog(factory = REFUND_FACTORY) {
  return render(
    <FactoriesLayoutContext.Provider
      value={{
        organizationId: "org-1",
        factoryId: PRIMARY_FACTORY_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        factory,
        factories: [factory],
        openCreateWorkOrder: vi.fn(),
      }}
    >
      <CreateWorkOrderDialog open onClose={vi.fn()} onCreated={vi.fn()} />
    </FactoriesLayoutContext.Provider>,
  );
}

describe("CreateWorkOrderDialog", () => {
  beforeEach(() => {
    permissions.isLoading = false;
    canAct.mockImplementation((resource: string, action: string) => resource === "work_orders" && action === "create");
    createMutate.mockReset();
    dispatchMutate.mockReset();
    meUser.current = null;
  });

  it("names the dialog New task instead of the fallback Dialog title", () => {
    renderDialog();

    expect(screen.getByRole("dialog", { name: "New task" })).toBeInTheDocument();
    expect(screen.queryByText("Dialog")).not.toBeInTheDocument();
  });

  it("keeps only expand and close controls in the header", () => {
    renderDialog();

    const header = screen.getByTestId("work-order-create-header");
    expect(within(header).getByTestId("work-order-create-fullscreen-button")).toBeInTheDocument();
    expect(within(header).queryByTestId("work-order-create-draft-button")).not.toBeInTheDocument();
  });

  it("renders Save as draft in the footer next to Start, with no line picker", () => {
    renderDialog();

    expect(screen.getByTestId("work-order-create-draft-button")).toBeInTheDocument();
    expect(screen.getByTestId("work-order-create-start")).toHaveTextContent("Start");
    expect(screen.queryByTestId("work-order-line-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-line-picker-panel")).not.toBeInTheDocument();
  });

  it("does not show an owner picker", () => {
    meUser.current = { id: "user-me", name: "Me" };
    renderDialog();

    expect(screen.queryByTestId("work-order-assignees-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mock-owner-pill")).not.toBeInTheDocument();
  });

  it("disables Start when the user cannot dispatch work orders", () => {
    renderDialog();

    expect(screen.getByTestId("work-order-create-start").closest(".pointer-events-none")).toBeInTheDocument();
  });

  it("keeps Start closed while permissions load", () => {
    permissions.isLoading = true;
    canAct.mockReturnValue(false);
    renderDialog();

    expect(screen.getByTestId("work-order-create-start").closest(".pointer-events-none")).toBeInTheDocument();
  });

  it("starts the work order on the first line", async () => {
    const user = userEvent.setup();
    canAct.mockReturnValue(true);
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });
    dispatchMutate.mockResolvedValue({});
    renderDialog();

    await user.type(screen.getByTestId("work-order-title-input"), "Ship the refunds line");
    await user.click(screen.getByTestId("work-order-create-start"));

    expect(screen.queryByTestId("work-order-line-picker-panel")).not.toBeInTheDocument();
    expect(dispatchMutate).toHaveBeenCalledWith({ orderId: "order-1", lineName: "plan-and-implement" });
  });

  it("disables Start when the workspace has no lines, while Save as draft stays enabled", async () => {
    canAct.mockReturnValue(true);
    renderDialog(EMPTY_FACTORY);

    await userEvent.setup().type(screen.getByTestId("work-order-title-input"), "Draft only");

    expect(screen.getByTestId("work-order-create-start")).toBeDisabled();
    expect(screen.getByTestId("work-order-create-draft-button")).not.toBeDisabled();
  });
});
