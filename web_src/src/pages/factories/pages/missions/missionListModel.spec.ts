import { describe, expect, it } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { buildWorkOrderListEntries } from "../../lib/workOrderListModel";
import {
  applyMissionFilter,
  applyMissionScope,
  assignWorkOrderToMission,
  buildMissionRailItems,
  findMissionById,
  resolveMissionForWorkOrder,
  resolveMissionDisplayStatus,
  resolveMissionProgress,
  resolveMissionStatus,
  visibleMissionItems,
} from "./missionListModel";
import { factoryMissionDetailPath } from "./missionPaths";
import {
  CHECKOUT_RELIABILITY_MISSION,
  REFUNDS_V2_MISSION,
  SEEDED_MISSIONS,
  missionByWorkOrderId,
} from "./missionMocks";

const factory: FactoriesFactory = { id: "f", name: "Refunds", key: "RF" };

function order(overrides: Partial<FactoriesWorkOrder> = {}): FactoriesWorkOrder {
  return {
    id: "wo-1",
    title: "Order title",
    state: "STATE_OPEN",
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    lineDispatches: [],
    ...overrides,
  };
}

describe("applyMissionFilter", () => {
  it("keeps tasks that belong to the selected mission", () => {
    const entries = buildWorkOrderListEntries(
      [order({ id: "wo-open-refunds" }), order({ id: "wo-draft-refunds" })],
      factory,
    );

    const filtered = applyMissionFilter(entries, CHECKOUT_RELIABILITY_MISSION.id, missionByWorkOrderId);

    expect(filtered.map((entry) => entry.id)).toEqual(["wo-open-refunds"]);
  });

  it("returns every entry when no mission is selected", () => {
    const entries = buildWorkOrderListEntries(
      [order({ id: "wo-open-refunds" }), order({ id: "wo-draft-refunds" })],
      factory,
    );

    expect(applyMissionFilter(entries, null, missionByWorkOrderId)).toEqual(entries);
  });
});

describe("resolveMissionProgress", () => {
  it("counts completed tasks in the mission and formats the label", () => {
    const entries = buildWorkOrderListEntries(
      [
        order({ id: "wo-a", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
        order({ id: "wo-b", state: "STATE_OPEN" }),
        order({ id: "wo-c", state: "STATE_CLOSED", result: "RESULT_FAILED" }),
        order({ id: "wo-other", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
      ],
      factory,
    );
    const byOrderId = { "wo-a": "mission-1", "wo-b": "mission-1", "wo-c": "mission-1" };

    const progress = resolveMissionProgress(entries, "mission-1", byOrderId);

    expect(progress).toEqual({
      completed: 1,
      total: 3,
      label: "1 of 3 completed",
      percent: 33,
    });
  });
});

describe("buildMissionRailItems", () => {
  it("builds one card per mission with progress from every member task", () => {
    const entries = buildWorkOrderListEntries(
      [
        order({ id: "wo-open-refunds" }),
        order({ id: "wo-closed-refunds", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
        order({ id: "wo-draft-refunds" }),
      ],
      factory,
    );

    const items = buildMissionRailItems(
      [CHECKOUT_RELIABILITY_MISSION, REFUNDS_V2_MISSION],
      entries,
      missionByWorkOrderId,
    );

    expect(items).toEqual([
      {
        mission: CHECKOUT_RELIABILITY_MISSION,
        progress: { completed: 1, total: 2, label: "1 of 2 completed", percent: 50 },
        status: "active",
      },
      {
        mission: REFUNDS_V2_MISSION,
        progress: { completed: 0, total: 0, label: "0 of 0 completed", percent: 0 },
        status: "draft",
      },
    ]);
  });

  it("uses a manual complete or close as the card status", () => {
    const entries = buildWorkOrderListEntries([order({ id: "wo-open-refunds" })], factory);

    const items = buildMissionRailItems(
      [CHECKOUT_RELIABILITY_MISSION, REFUNDS_V2_MISSION],
      entries,
      missionByWorkOrderId,
      {
        [CHECKOUT_RELIABILITY_MISSION.id]: "completed",
        [REFUNDS_V2_MISSION.id]: "closed",
      },
    );

    expect(items.map((item) => item.status)).toEqual(["completed", "closed"]);
  });
});

describe("visibleMissionItems", () => {
  function item(name: string, total: number): ReturnType<typeof buildMissionRailItems>[number] {
    return {
      mission: {
        id: `mission-${name}`,
        name,
        description: "",
        assignee: { id: "user-1", name: "Alex Reviewer" },
      },
      progress: { completed: 0, total, label: `0 of ${total} completed`, percent: 0 },
      status: total === 0 ? "draft" : "active",
    };
  }

  it("shows missions with work first and hides the rest until expanded", () => {
    const items = [item("Empty A", 0), item("Busy", 3), item("Empty B", 0), item("Also busy", 1), item("Empty C", 0)];

    const collapsed = visibleMissionItems(items, false);
    expect(collapsed.visible.map((entry) => entry.mission.name)).toEqual(["Busy", "Also busy"]);
    expect(collapsed.hiddenCount).toBe(3);

    const expanded = visibleMissionItems(items, true);
    expect(expanded.visible.map((entry) => entry.mission.name)).toEqual([
      "Busy",
      "Also busy",
      "Empty A",
      "Empty B",
      "Empty C",
    ]);
    expect(expanded.hiddenCount).toBe(0);
  });

  it("caps the default list when many missions have work", () => {
    const items = ["A", "B", "C", "D", "E"].map((name, index) => item(name, 5 - index));
    const collapsed = visibleMissionItems(items, false);

    expect(collapsed.visible.map((entry) => entry.mission.name)).toEqual(["A", "B", "C", "D"]);
    expect(collapsed.hiddenCount).toBe(1);
  });
});

describe("seeded missions", () => {
  it("keeps enough missions to exercise a dense list", () => {
    expect(SEEDED_MISSIONS.length).toBeGreaterThanOrEqual(10);
    expect(SEEDED_MISSIONS.every((mission) => mission.assignee.id && mission.assignee.name)).toBe(true);
  });
});

describe("findMissionById", () => {
  it("returns the mission or null", () => {
    expect(findMissionById(SEEDED_MISSIONS, CHECKOUT_RELIABILITY_MISSION.id)).toEqual(CHECKOUT_RELIABILITY_MISSION);
    expect(findMissionById(SEEDED_MISSIONS, "missing")).toBeNull();
  });
});

describe("factoryMissionDetailPath", () => {
  it("nests the mission under the Missions list path", () => {
    expect(factoryMissionDetailPath("org-1", "RF", CHECKOUT_RELIABILITY_MISSION.id)).toBe(
      `/org-1/workspaces/RF/missions/${CHECKOUT_RELIABILITY_MISSION.id}`,
    );
  });
});

describe("assignWorkOrderToMission", () => {
  it("assigns a task to a mission without mutating the source map", () => {
    const source = { "wo-draft-refunds": CHECKOUT_RELIABILITY_MISSION.id };

    const next = assignWorkOrderToMission(source, "wo-open-refunds", REFUNDS_V2_MISSION.id);

    expect(next).toEqual({
      "wo-draft-refunds": CHECKOUT_RELIABILITY_MISSION.id,
      "wo-open-refunds": REFUNDS_V2_MISSION.id,
    });
    expect(source).toEqual({ "wo-draft-refunds": CHECKOUT_RELIABILITY_MISSION.id });
  });

  it("moves a task from one mission to another", () => {
    const next = assignWorkOrderToMission(
      { "wo-open-refunds": CHECKOUT_RELIABILITY_MISSION.id },
      "wo-open-refunds",
      REFUNDS_V2_MISSION.id,
    );

    expect(next).toEqual({ "wo-open-refunds": REFUNDS_V2_MISSION.id });
  });

  it("removes the task from every mission when the mission is cleared", () => {
    const next = assignWorkOrderToMission(
      {
        "wo-open-refunds": CHECKOUT_RELIABILITY_MISSION.id,
        "wo-draft-refunds": REFUNDS_V2_MISSION.id,
      },
      "wo-open-refunds",
      null,
    );

    expect(next).toEqual({ "wo-draft-refunds": REFUNDS_V2_MISSION.id });
  });
});

describe("resolveMissionStatus", () => {
  it("is draft when the mission has no tasks", () => {
    const entries = buildWorkOrderListEntries([order({ id: "wo-other" })], factory);

    expect(resolveMissionStatus(entries, "mission-1", {})).toBe("draft");
  });

  it("is active when any member is running or waiting", () => {
    const entries = buildWorkOrderListEntries(
      [
        order({
          id: "wo-a",
          state: "STATE_OPEN",
          lineDispatches: [
            {
              id: "dispatch-1",
              state: "STATE_ACTIVE",
              stepExecutions: [{ id: "exec-1", state: "STATE_STARTED", step: "plan" }],
            },
          ],
        }),
        order({ id: "wo-b", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
      ],
      factory,
    );

    expect(resolveMissionStatus(entries, "mission-1", { "wo-a": "mission-1", "wo-b": "mission-1" })).toBe("active");
  });

  it("is active when assigned work is still open or draft", () => {
    const entries = buildWorkOrderListEntries(
      [order({ id: "wo-a", state: "STATE_OPEN" }), order({ id: "wo-b", state: "STATE_DRAFT" })],
      factory,
    );

    expect(resolveMissionStatus(entries, "mission-1", { "wo-a": "mission-1", "wo-b": "mission-1" })).toBe("active");
  });

  it("is completed when every member is completed or failed", () => {
    const entries = buildWorkOrderListEntries(
      [
        order({ id: "wo-a", state: "STATE_CLOSED", result: "RESULT_COMPLETED" }),
        order({ id: "wo-b", state: "STATE_CLOSED", result: "RESULT_FAILED" }),
      ],
      factory,
    );

    expect(resolveMissionStatus(entries, "mission-1", { "wo-a": "mission-1", "wo-b": "mission-1" })).toBe("completed");
  });
});

describe("resolveMissionDisplayStatus", () => {
  it("keeps the work-order status when the mission is not closed by hand", () => {
    expect(resolveMissionDisplayStatus("active")).toBe("active");
    expect(resolveMissionDisplayStatus("draft")).toBe("draft");
  });

  it("shows Completed after a manual complete, even when work is still open", () => {
    expect(resolveMissionDisplayStatus("active", "completed")).toBe("completed");
  });

  it("shows Closed after a manual close, even when work is still open", () => {
    expect(resolveMissionDisplayStatus("active", "closed")).toBe("closed");
  });
});

describe("applyMissionScope", () => {
  function item(name: string, status: "draft" | "active" | "completed", assigneeId = "user-1") {
    return {
      mission: {
        id: `mission-${name}`,
        name,
        description: "",
        assignee: { id: assigneeId, name: "Alex Reviewer" },
      },
      progress: { completed: 0, total: status === "draft" ? 0 : 1, label: "0 of 1 completed", percent: 0 },
      status,
    };
  }

  const items = [item("Draft", "draft"), item("Active", "active"), item("Done", "completed")];

  it("hides completed and closed missions when the page scope is Active", () => {
    const closed = { "mission-Active": "closed" as const };

    expect(applyMissionScope(items, "active", {}).map((entry) => entry.mission.name)).toEqual(["Draft", "Active"]);
    expect(applyMissionScope(items, "active", closed).map((entry) => entry.mission.name)).toEqual(["Draft"]);
  });

  it("keeps every mission when the page scope is All", () => {
    expect(applyMissionScope(items, "all", { "mission-Active": "closed" }).map((entry) => entry.mission.name)).toEqual([
      "Draft",
      "Active",
      "Done",
    ]);
  });

  it("keeps missions assigned to the current user when the page scope is My", () => {
    const mixed = [
      item("Mine", "active", "user-me"),
      item("Theirs", "active", "user-other"),
      item("Done mine", "completed", "user-me"),
    ];

    expect(applyMissionScope(mixed, "my", {}, "user-me").map((entry) => entry.mission.name)).toEqual([
      "Mine",
      "Done mine",
    ]);
  });
});

describe("resolveMissionForWorkOrder", () => {
  it("returns the assigned mission or null", () => {
    expect(resolveMissionForWorkOrder(SEEDED_MISSIONS, missionByWorkOrderId, "wo-open-refunds")).toEqual(
      CHECKOUT_RELIABILITY_MISSION,
    );
    expect(resolveMissionForWorkOrder(SEEDED_MISSIONS, missionByWorkOrderId, "wo-draft-refunds")).toBeNull();
  });
});
