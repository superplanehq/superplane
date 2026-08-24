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
import type * as CreateWorkOrderPropertyPillsModule from "./CreateWorkOrderPropertyPills";
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

vi.mock("./CreateWorkOrderPropertyPills", async () => {
  const actual = await vi.importActual<typeof CreateWorkOrderPropertyPillsModule>("./CreateWorkOrderPropertyPills");
  return {
    ...actual,
    CreateWorkOrderPropertyPills: (props: { assigneeIds: string[] }) => (
      <div data-testid="mock-owner-pill">{props.assigneeIds.join(",") || "no owner"}</div>
    ),
  };
});

function renderDialog(factory = REFUND_FACTORY, initialTitle?: string) {
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
      <CreateWorkOrderDialog open initialTitle={initialTitle} onClose={vi.fn()} onCreated={vi.fn()} />
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

  it("names the dialog New work order instead of the fallback Dialog title", () => {
    renderDialog();

    expect(screen.getByRole("dialog", { name: "New work order" })).toBeInTheDocument();
    expect(screen.queryByText("Dialog")).not.toBeInTheDocument();
  });

  it("seeds the title from the surface that opened it", () => {
    renderDialog(REFUND_FACTORY, "Fix the billing export");

    expect(screen.getByTestId("work-order-title-input")).toHaveValue("Fix the billing export");
  });

  it("keeps only expand and close controls in the header", () => {
    renderDialog();

    const header = screen.getByTestId("work-order-create-header");
    expect(within(header).getByTestId("work-order-create-fullscreen-button")).toBeInTheDocument();
    expect(within(header).queryByTestId("work-order-create-draft-button")).not.toBeInTheDocument();
  });

  it("renders Save as draft in the footer next to Send to line, with no separate Line control", () => {
    renderDialog();

    expect(screen.getByTestId("work-order-create-draft-button")).toBeInTheDocument();
    expect(screen.getByTestId("work-order-create-send-to-line")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-line-button")).not.toBeInTheDocument();
  });

  it("defaults Owner to the current user without any click", () => {
    meUser.current = { id: "user-me", name: "Me" };
    renderDialog();

    expect(screen.getByTestId("mock-owner-pill")).toHaveTextContent("user-me");
  });

  it("leaves Owner empty when there is no current user", () => {
    renderDialog();

    expect(screen.getByTestId("mock-owner-pill")).toHaveTextContent("no owner");
  });

  it("disables Send to line when the user cannot dispatch work orders", () => {
    renderDialog();

    expect(screen.getByTestId("work-order-create-send-to-line").closest(".pointer-events-none")).toBeInTheDocument();
  });

  it("keeps Send to line closed while permissions load", () => {
    permissions.isLoading = true;
    canAct.mockReturnValue(false);
    renderDialog();

    expect(screen.getByTestId("work-order-create-send-to-line").closest(".pointer-events-none")).toBeInTheDocument();
  });

  it("opens the line list and dispatches immediately when a line is picked", async () => {
    const user = userEvent.setup();
    canAct.mockReturnValue(true);
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });
    dispatchMutate.mockResolvedValue({});
    renderDialog();

    await user.type(screen.getByTestId("work-order-title-input"), "Ship the refunds line");
    await user.click(screen.getByTestId("work-order-create-send-to-line"));

    const panel = screen.getByTestId("work-order-line-picker-panel");
    expect(panel).toHaveAttribute("data-align", "end");

    await user.click(screen.getByTestId("work-order-line-option-plan-and-implement"));

    expect(dispatchMutate).toHaveBeenCalledWith({ orderId: "order-1", lineName: "plan-and-implement" });
  });

  it("disables Send to line with a tooltip when the workspace has no lines, while Save as draft stays enabled", async () => {
    canAct.mockReturnValue(true);
    renderDialog(EMPTY_FACTORY);

    await userEvent.setup().type(screen.getByTestId("work-order-title-input"), "Draft only");

    expect(screen.getByTestId("work-order-create-send-to-line")).toBeDisabled();
    expect(screen.getByTestId("work-order-create-draft-button")).not.toBeDisabled();
  });
});
