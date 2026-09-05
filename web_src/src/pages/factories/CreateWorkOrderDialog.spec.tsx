import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  EMPTY_FACTORY,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
} from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { FactoriesLayoutContext } from "./layout/factoriesLayoutContext";

const { createMutate, dispatchMutate, meUser } = vi.hoisted(() => ({
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
    createMutate.mockReset();
    dispatchMutate.mockReset();
    meUser.current = null;
  });

  afterEach(async () => {
    // The dialog mounts a Radix focus scope that schedules a `setTimeout(0)` on
    // unmount to dispatch its "auto focus on unmount" event. Unmount here and
    // flush that timer while jsdom is still alive; otherwise it fires during
    // environment teardown and throws an unhandled "dispatchEvent" TypeError
    // that fails the whole test shard even though every test passed.
    cleanup();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
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
    expect(within(header).queryByTestId("work-order-create-button")).not.toBeInTheDocument();
  });

  it("renders a single Create button in the footer, with no line picker", () => {
    renderDialog();

    expect(screen.getByTestId("work-order-create-button")).toHaveTextContent("Create");
    expect(screen.queryByTestId("work-order-create-start")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-line-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-line-picker-panel")).not.toBeInTheDocument();
  });

  it("does not show an owner picker", () => {
    meUser.current = { id: "user-me", name: "Me" };
    renderDialog();

    expect(screen.queryByTestId("work-order-assignees-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mock-owner-pill")).not.toBeInTheDocument();
  });

  it("creates the task without sending it to a line", async () => {
    const user = userEvent.setup();
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });
    renderDialog();

    await user.type(screen.getByTestId("work-order-title-input"), "Ship the refunds line");
    await user.click(screen.getByTestId("work-order-create-button"));

    expect(createMutate).toHaveBeenCalled();
    expect(dispatchMutate).not.toHaveBeenCalled();
  });

  it("keeps Create enabled when the workspace has no lines", async () => {
    renderDialog(EMPTY_FACTORY);

    await userEvent.setup().type(screen.getByTestId("work-order-title-input"), "Draft only");

    expect(screen.getByTestId("work-order-create-button")).not.toBeDisabled();
  });
});
