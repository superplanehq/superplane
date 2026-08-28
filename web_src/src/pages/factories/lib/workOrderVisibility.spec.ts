import { describe, expect, it } from "vitest";

import { visibleWorkOrdersForCollections } from "./workOrderVisibility";

describe("visibleWorkOrdersForCollections", () => {
  it("excludes intake work orders from normal list and board collections", () => {
    const workOrders = [
      { id: "work-order-intake", state: "STATE_INTAKE" },
      { id: "work-order-draft", state: "STATE_DRAFT" },
      { id: "work-order-open", state: "STATE_OPEN" },
    ];

    expect(visibleWorkOrdersForCollections(workOrders).map((workOrder) => workOrder.id)).toEqual([
      "work-order-draft",
      "work-order-open",
    ]);
  });
});
