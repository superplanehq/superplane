import { describe, expect, it } from "bun:test";

import { createWithAgentViewFromSession } from "./planningSessionView";

const EXTRAS = { composer: "", right: { kind: "empty" as const }, endConfirmOpen: false };

describe("task messages in the planning session view", () => {
  it("maps a task message to a card that follows the live created order", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        executionId: "exec-1",
        draft: { title: "Big task", description: "", workOrderId: "wo-parent" },
        created: [
          { id: "wo-parent", key: "NEW-1", title: "Big task", number: "1" },
          { id: "wo-2", key: "NEW-2", title: "Renamed later", number: "2" },
        ],
        messages: [
          {
            id: "msg-task",
            role: "task",
            text: JSON.stringify({ work_order_id: "wo-2", key: "NEW-2", title: "Add the retry table" }),
            createdAt: "2026-09-18T10:00:00Z",
            activityId: "activity-1",
          },
          { id: "msg-bad", role: "task", text: "not json" },
        ],
      },
      EXTRAS,
    );

    expect(view.messages).toEqual([
      {
        id: "msg-task",
        kind: "task",
        role: "task",
        workOrderId: "wo-2",
        key: "NEW-2",
        title: "Renamed later",
        number: 2,
        activityId: "activity-1",
        createdAtMs: Date.parse("2026-09-18T10:00:00Z"),
      },
    ]);
  });

  it("falls back to the message snapshot when the created order is gone", () => {
    const view = createWithAgentViewFromSession(
      {
        repository: "acme/payments",
        canvasId: "canvas-1",
        executionId: "exec-1",
        messages: [
          {
            id: "msg-task",
            role: "task",
            text: JSON.stringify({ work_order_id: "wo-2", key: "NEW-2", title: "Add the retry table" }),
          },
        ],
      },
      EXTRAS,
    );

    expect(view.messages).toEqual([
      { id: "msg-task", kind: "task", role: "task", workOrderId: "wo-2", key: "NEW-2", title: "Add the retry table" },
    ]);
  });
});
