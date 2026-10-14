import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import type { BuiltInAgentTask } from "./builtInAgentMocks";
import { useBuiltInAgentPrototype } from "./useBuiltInAgentPrototype";

function task(overrides: Partial<BuiltInAgentTask> = {}): BuiltInAgentTask {
  return {
    id: "task-old",
    title: "Reconcile",
    state: "draft",
    lane: "backlog",
    owner: null,
    updatedAt: "2026-09-17T00:00:00.000Z",
    ...overrides,
  };
}

describe("useBuiltInAgentPrototype", () => {
  it("clears a pending plan when a task is created", () => {
    const existing = task();
    const { result } = renderHook(() =>
      useBuiltInAgentPrototype({
        tasks: [existing],
        pendingPlan: { taskIds: [existing.id], reason: "Do waiting work first." },
      }),
    );

    act(() => {
      result.current.createTask("Review ledger");
    });

    expect(result.current.pendingPlan).toBeNull();
    expect(result.current.tasks.map((item) => item.title)).toEqual(["Review ledger", "Reconcile"]);
  });

  it("clears a pending plan when chat creates a task", () => {
    const existing = task();
    const { result } = renderHook(() =>
      useBuiltInAgentPrototype({
        tasks: [existing],
        pendingPlan: { taskIds: [existing.id], reason: "Do waiting work first." },
      }),
    );

    act(() => {
      result.current.sendMessage("create Review ledger");
    });

    expect(result.current.pendingPlan).toBeNull();
    expect(result.current.tasks.some((item) => item.title === "Review ledger")).toBe(true);
  });
});
