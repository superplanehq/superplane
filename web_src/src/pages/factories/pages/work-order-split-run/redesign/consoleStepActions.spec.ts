import { describe, expect, it } from "bun:test";

import type { FactoriesWorkOrderEvent } from "@/api-client";
import { stepActionRows } from "./consoleStepActions";

function event(type: string, timestamp: string, payload: Record<string, unknown>): FactoriesWorkOrderEvent {
  return { type, timestamp, event: payload };
}

const EVENTS = [
  event("order.assignees.updated", "2026-01-03T00:00:00Z", {
    run: { id: "run-other" },
    assigned: [{ id: "user-ada" }],
    reason: "Other run.",
  }),
  event("order.assignees.updated", "2026-01-02T00:00:00Z", {
    run: { id: "run-1" },
    assigned: [{ id: "user-ada" }],
    reason: "Changed pkg/payments.",
  }),
  event("order.check.reported", "2026-01-01T00:00:00Z", {
    run: { id: "run-1" },
    check: { name: "Risk score", score: 3, maxScore: 5, format: "fraction", summary: "Low risk overall." },
  }),
  event("order.assignees.updated", "2026-01-04T00:00:00Z", {
    run: { id: "run-1" },
    assigned: [{ id: "user-bea" }],
    reason: "   ",
  }),
];

describe("stepActionRows", () => {
  it("keeps this run, oldest first, and drops an empty reason", () => {
    const rows = stepActionRows(EVENTS, new Set(["run-1"]), (userId) =>
      userId === "user-ada" ? "Ada" : userId === "user-bea" ? "Bea" : undefined,
    );

    expect(rows.map((row) => row.title)).toEqual([
      "Reported Risk score 3/5",
      "Assigned Ada as owner",
      "Assigned Bea as owner",
    ]);
    expect(rows.map((row) => row.detail)).toEqual(["Low risk overall.", "Changed pkg/payments.", undefined]);
    expect(rows.some((row) => row.detail === "Other run.")).toBe(false);
  });
});
