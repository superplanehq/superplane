import { describe, expect, it } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import {
  buildAssigneeFilterOptions,
  buildLineFilterOptions,
  buildStatusFilterOptions,
  buildWorkOrderFilterChips,
} from "./workOrderFilterOptions";
import { EMPTY_WORK_ORDER_FILTERS, UNASSIGNED_FILTER_VALUE, buildWorkOrderListEntries } from "./workOrderListModel";

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

describe("buildStatusFilterOptions", () => {
  it("lists every display status with its dot class", () => {
    const options = buildStatusFilterOptions();
    expect(options.map((option) => option.value)).toEqual([
      "draft",
      "running",
      "waiting",
      "completed",
      "failed",
      "cancelled",
    ]);
    expect(options.every((option) => Boolean(option.dot))).toBe(true);
  });
});

describe("buildLineFilterOptions", () => {
  it("skips lines with no id and names untitled lines", () => {
    const options = buildLineFilterOptions([
      { id: "line-a", name: "hotfix" },
      { id: "line-b", name: "   " },
      { name: "orphan" },
    ]);

    expect(options).toEqual([
      { value: "line-a", label: "hotfix" },
      { value: "line-b", label: "Untitled line" },
    ]);
  });
});

describe("buildAssigneeFilterOptions", () => {
  it("puts No Owner first, then people sorted by name", () => {
    const entries = buildWorkOrderListEntries(
      [
        order({ id: "wo-1", assignees: [{ id: "u2", name: "Zoe" }] }),
        order({
          id: "wo-2",
          assignees: [
            { id: "u1", name: "Alex" },
            { id: "u2", name: "Zoe" },
          ],
        }),
        order({ id: "wo-3" }),
      ],
      factory,
    );

    expect(buildAssigneeFilterOptions(entries)).toEqual([
      { value: UNASSIGNED_FILTER_VALUE, label: "No Owner" },
      { value: "u1", label: "Alex" },
      { value: "u2", label: "Zoe" },
    ]);
  });
});

describe("buildWorkOrderFilterChips", () => {
  const options = {
    lines: [{ value: "line-a", label: "hotfix" }],
    assignees: [
      { value: UNASSIGNED_FILTER_VALUE, label: "No Owner" },
      { value: "u1", label: "Alex" },
    ],
  };

  it("labels one chip per applied filter", () => {
    const chips = buildWorkOrderFilterChips(
      { statuses: ["running"], lineIds: ["line-a"], assigneeIds: ["u1", UNASSIGNED_FILTER_VALUE] },
      options,
    );

    expect(chips.map((chip) => chip.label)).toEqual([
      "Status is Running",
      "Line is hotfix",
      "Owner is Alex",
      "No Owner",
    ]);
  });

  it("returns nothing when no filter is applied", () => {
    expect(buildWorkOrderFilterChips(EMPTY_WORK_ORDER_FILTERS, options)).toEqual([]);
  });

  it("falls back to a placeholder when an option disappeared", () => {
    const chips = buildWorkOrderFilterChips({ ...EMPTY_WORK_ORDER_FILTERS, lineIds: ["deleted-line"] }, options);
    expect(chips[0].label).toBe("Line is Unknown line");
  });
});
