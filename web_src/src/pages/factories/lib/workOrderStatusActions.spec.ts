import { describe, expect, it } from "vitest";

import { buildWorkOrderStatusActions, type WorkOrderStatusActionInput } from "./workOrderStatusActions";

function labelsOf(overrides: Partial<WorkOrderStatusActionInput> = {}) {
  return buildWorkOrderStatusActions({
    displayStatus: "waiting",
    isOpen: true,
    isDispatchable: true,
    isClosed: false,
    canClose: true,
    canManage: true,
    isClosing: false,
    isUpdatingStatus: false,
    ...overrides,
  }).map((action) => action.label);
}

describe("buildWorkOrderStatusActions", () => {
  it("offers Complete, Reject, and Back to draft for an open waiting order", () => {
    expect(labelsOf({ displayStatus: "waiting", isOpen: true })).toEqual(["Complete", "Reject", "Back to draft"]);
  });

  it("hides Back to draft while a line is running", () => {
    expect(labelsOf({ displayStatus: "running", isOpen: true })).toEqual(["Complete", "Reject"]);
  });

  it("offers Reject for a draft order", () => {
    expect(
      labelsOf({
        displayStatus: "draft",
        isOpen: false,
        isDispatchable: true,
        isClosed: false,
      }),
    ).toEqual(["Reject"]);
  });

  it("offers Reopen for a closed order", () => {
    expect(
      labelsOf({
        displayStatus: "completed",
        isOpen: false,
        isDispatchable: false,
        isClosed: true,
      }),
    ).toEqual(["Reopen"]);
  });
});
