import { afterEach, describe, expect, it, vi } from "bun:test";

import { findPlanningSessionByWorkOrder, sendPlanningSessionMessage } from "./planningSessionClient";

describe("planning session errors", () => {
  afterEach(() => vi.restoreAllMocks());

  it.each([
    "Subscribe to a plan to run agents.",
    "This organization has no hosted credit.",
    "This workspace has no remaining hosted credit.",
  ])("preserves the reason a message was rejected: %s", async (message) => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(Response.json({ code: 9, message }, { status: 400 }));

    await expect(sendPlanningSessionMessage("org-1", "factory-1", "session-1", "Continue")).rejects.toThrow(message);
  });

  it("preserves the reason a session could not load", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      Response.json({ code: 7, message: "You do not have permission to view this task." }, { status: 403 }),
    );

    await expect(findPlanningSessionByWorkOrder("org-1", "factory-1", "order-1")).rejects.toThrow(
      "You do not have permission to view this task.",
    );
  });
});
