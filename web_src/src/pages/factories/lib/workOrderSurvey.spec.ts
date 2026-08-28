import { describe, expect, it } from "vitest";
import type { FactoriesWorkOrder } from "@/api-client";

import { workOrderPendingSurvey } from "./workOrderSurvey";

function order(pendingSurvey: unknown): FactoriesWorkOrder {
  return { id: "wo-1", title: "Ship it", pendingSurvey } as FactoriesWorkOrder;
}

describe("workOrderPendingSurvey", () => {
  it("returns a pending survey with questions", () => {
    expect(
      workOrderPendingSurvey(
        order({
          id: "s-1",
          status: "pending",
          questions: [{ id: "scope", prompt: "Where?", options: ["A", "B"], allowFreeText: true }],
        }),
      ),
    ).toEqual({
      id: "s-1",
      status: "pending",
      questions: [{ id: "scope", prompt: "Where?", options: ["A", "B"], allowFreeText: true }],
    });
  });

  it("hides answered or empty surveys", () => {
    expect(
      workOrderPendingSurvey(order({ id: "s-1", status: "answered", questions: [{ id: "x", prompt: "X" }] })),
    ).toBeUndefined();
    expect(workOrderPendingSurvey(order({ id: "s-1", status: "pending", questions: [] }))).toBeUndefined();
    expect(workOrderPendingSurvey({ id: "wo-1", title: "Ship it" } as FactoriesWorkOrder)).toBeUndefined();
  });
});
