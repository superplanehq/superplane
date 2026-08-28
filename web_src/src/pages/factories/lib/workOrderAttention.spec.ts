import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrder } from "@/api-client";

import { getWorkOrderAttentionReason, WORK_ORDER_ATTENTION_LABEL } from "./workOrderAttention";

function order(overrides: Partial<FactoriesWorkOrder> = {}): FactoriesWorkOrder {
  return {
    id: "wo-1",
    title: "Order",
    state: "STATE_OPEN",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    lineDispatches: [],
    ...overrides,
  };
}

describe("getWorkOrderAttentionReason", () => {
  it("returns null when the work order is not waiting or failed", () => {
    expect(getWorkOrderAttentionReason(order({ state: "STATE_DRAFT" }))).toBeNull();
  });

  it("labels a closed failed work order as Run failed", () => {
    expect(getWorkOrderAttentionReason(order({ state: "STATE_CLOSED", result: "RESULT_FAILED" }))).toBe("failed");
  });

  it("labels a failed latest step as Run failed", () => {
    expect(
      getWorkOrderAttentionReason(
        order({
          lineDispatches: [
            {
              id: "d1",
              state: "STATE_FINISHED",
              stepExecutions: [
                {
                  id: "e1",
                  step: "implement",
                  state: "STATE_FINISHED",
                  result: "RESULT_FAILED",
                  updatedAt: "2024-06-02T00:00:00Z",
                },
              ],
            },
          ],
        }),
      ),
    ).toBe("failed");
    expect(WORK_ORDER_ATTENTION_LABEL.failed).toBe("Run failed");
  });

  it("labels a visible status note as Waiting for user review", () => {
    expect(
      getWorkOrderAttentionReason(
        order({
          statusNotes: [{ key: "pr-closure", headline: "Review the pull request", body: "PR #12 is open." }],
        }),
      ),
    ).toBe("approval");
    expect(
      getWorkOrderAttentionReason(
        order({
          statusNotes: [{ key: "decision", headline: "Confirm the cutover window", body: "Pick a date." }],
        }),
      ),
    ).toBe("approval");
    expect(
      getWorkOrderAttentionReason(
        order({
          statusNotes: [
            {
              key: "pr-closure",
              headline: "Waiting for user review",
              body: "The pull request is open.",
            },
          ],
        }),
      ),
    ).toBe("approval");
    expect(
      getWorkOrderAttentionReason(
        order({
          statusNotes: [{ key: "agent-question", headline: "The agent has a question", body: "Which provider?" }],
        }),
      ),
    ).toBe("approval");
    expect(WORK_ORDER_ATTENTION_LABEL.approval).toBe("Waiting for user review");
  });

  it("labels an active PR-feedback run as Addressing user feedback", () => {
    expect(
      getWorkOrderAttentionReason(
        order({
          statusNotes: [
            {
              key: "pr-closure",
              headline: "Waiting for user review",
              body: "Tag `@superplaneagent` to request changes.",
            },
          ],
        }),
        { addressingFeedback: true },
      ),
    ).toBe("feedback");
    expect(WORK_ORDER_ATTENTION_LABEL.feedback).toBe("Addressing user feedback");
  });

  it("labels a cancelled latest step as Stopped", () => {
    expect(
      getWorkOrderAttentionReason(
        order({
          lineDispatches: [
            {
              id: "d1",
              state: "STATE_FINISHED",
              stepExecutions: [
                {
                  id: "e1",
                  step: "implement",
                  state: "STATE_FINISHED",
                  result: "RESULT_CANCELLED",
                  updatedAt: "2024-06-02T00:00:00Z",
                },
              ],
            },
          ],
        }),
      ),
    ).toBe("stopped");
    expect(WORK_ORDER_ATTENTION_LABEL.stopped).toBe("Stopped");
  });

  it("labels idle waiting work as Needs attention", () => {
    expect(getWorkOrderAttentionReason(order())).toBe("stalled");
    expect(WORK_ORDER_ATTENTION_LABEL.stalled).toBe("Needs attention");
  });
});
