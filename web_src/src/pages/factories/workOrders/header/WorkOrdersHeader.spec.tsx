import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { useWorkOrderListState } from "../../lib/useWorkOrderListState";
import { WorkOrdersHeader } from "./WorkOrdersHeader";

function HeaderHarness({
  onCreateWorkOrder,
  canCreate = true,
}: {
  onCreateWorkOrder: () => void;
  canCreate?: boolean;
}) {
  const state = useWorkOrderListState();
  return (
    <WorkOrdersHeader
      state={state}
      entries={[]}
      factoryLines={[]}
      onCreateWorkOrder={onCreateWorkOrder}
      canCreate={canCreate}
      permissionsLoading={false}
    />
  );
}

describe("WorkOrdersHeader", () => {
  it("opens the create dialog in place instead of changing the path", async () => {
    const user = userEvent.setup();
    const onCreateWorkOrder = vi.fn();

    render(<HeaderHarness onCreateWorkOrder={onCreateWorkOrder} />);

    await user.click(screen.getByTestId("work-order-list-create-button"));

    expect(onCreateWorkOrder).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("work-order-list-create-button")).not.toHaveAttribute("href");
  });
});
