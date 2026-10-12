import { describe, expect, it } from "bun:test";
import type { FactoriesWorkOrder } from "@/api-client";

import { collectLineBacklogOrders } from "./linePhaseRuns";

describe("collectLineBacklogOrders order", () => {
  it("does not resort drafts when updated_at changes in place", () => {
    const older: FactoriesWorkOrder = {
      id: "wo-old",
      title: "Older draft",
      state: "STATE_DRAFT",
      updatedAt: "2026-01-01T00:00:00.000Z",
      lineDispatches: [],
    };
    const newer: FactoriesWorkOrder = {
      id: "wo-new",
      title: "Newer draft",
      state: "STATE_DRAFT",
      updatedAt: "2026-02-01T00:00:00.000Z",
      lineDispatches: [],
    };
    const bumpedMiddle = { ...newer, updatedAt: "2026-03-01T00:00:00.000Z" };

    expect(collectLineBacklogOrders([older, newer]).map((entry) => entry.id)).toEqual(["wo-old", "wo-new"]);
    expect(collectLineBacklogOrders([older, bumpedMiddle]).map((entry) => entry.id)).toEqual(["wo-old", "wo-new"]);
  });
});
