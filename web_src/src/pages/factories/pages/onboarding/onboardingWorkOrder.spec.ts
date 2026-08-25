import { describe, expect, it, vi } from "vitest";

import { createAndDispatchInitialWorkOrder } from "./onboardingWorkOrder";

describe("createAndDispatchInitialWorkOrder", () => {
  it("dispatches the created work order to the provisioned line", async () => {
    const createWorkOrder = vi.fn().mockResolvedValue({ id: "order-1", number: 42 });
    const dispatchWorkOrder = vi.fn().mockResolvedValue({});

    const order = await createAndDispatchInitialWorkOrder({
      title: "Improve AGENTS.md",
      description: "Document the repository conventions.",
      lineName: "plan-and-implement",
      createWorkOrder,
      dispatchWorkOrder,
    });

    expect(createWorkOrder).toHaveBeenCalledWith({
      title: "Improve AGENTS.md",
      description: "Document the repository conventions.",
    });
    expect(dispatchWorkOrder).toHaveBeenCalledWith({
      orderId: "order-1",
      lineName: "plan-and-implement",
    });
    expect(order).toEqual({ id: "order-1", number: 42 });
  });

  it("does not dispatch when the created work order has no ID", async () => {
    const dispatchWorkOrder = vi.fn();

    await expect(
      createAndDispatchInitialWorkOrder({
        title: "Improve AGENTS.md",
        description: "Document the repository conventions.",
        lineName: "plan-and-implement",
        createWorkOrder: vi.fn().mockResolvedValue({ number: 42 }),
        dispatchWorkOrder,
      }),
    ).rejects.toThrow("Created work order has no ID");

    expect(dispatchWorkOrder).not.toHaveBeenCalled();
  });
});
