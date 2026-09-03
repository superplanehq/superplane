import { describe, expect, it } from "vitest";

import {
  buildSpendingReport,
  EMPTY_SPENDING_FILTERS,
  filterSpendingEvents,
  formatFilterTriggerLabel,
  formatSpendingRangeCaption,
  hasActiveSpendingFilters,
  modelKey,
  rangeForPreset,
  rangeFromCustomDays,
  toggleFilterValue,
  type SpendingCatalogs,
  type SpendingUsageEvent,
} from "./spendingRedesignLib";

const NOW = new Date("2026-09-03T12:00:00.000Z");

const catalogs: SpendingCatalogs = {
  users: [
    { id: "user-a", label: "Alex" },
    { id: "user-b", label: "Jamie" },
  ],
  workspaces: [
    { id: "ws-refunds", label: "Semaphore" },
    { id: "ws-payments", label: "SuperPlane" },
  ],
  models: [
    { id: "anthropic/claude-sonnet-4-6", label: "claude-sonnet-4-6" },
    { id: "openai/gpt-4o", label: "gpt-4o" },
  ],
  machines: [
    { id: "e1-large-amd64", label: "e1-large-amd64" },
    { id: "e1-tiny-amd64", label: "e1-tiny-amd64" },
  ],
};

function event(
  overrides: Partial<SpendingUsageEvent> & Pick<SpendingUsageEvent, "id" | "occurredAt">,
): SpendingUsageEvent {
  return {
    factoryId: "ws-refunds",
    userId: "user-a",
    provider: "anthropic",
    model: "claude-sonnet-4-6",
    usageKind: "model",
    fundingSource: "hosted",
    machineType: "",
    totalTokens: 1000,
    durationSeconds: 0,
    costCents: 100,
    ...overrides,
  };
}

const ledger: SpendingUsageEvent[] = [
  event({ id: "today-model", occurredAt: "2026-09-03T10:00:00.000Z", costCents: 250, totalTokens: 4000 }),
  event({
    id: "today-compute",
    occurredAt: "2026-09-03T09:00:00.000Z",
    usageKind: "compute",
    provider: "runner",
    model: "e1-large-amd64",
    machineType: "e1-large-amd64",
    totalTokens: 0,
    durationSeconds: 600,
    costCents: 40,
  }),
  event({
    id: "week-gpt",
    occurredAt: "2026-08-30T15:00:00.000Z",
    provider: "openai",
    model: "gpt-4o",
    userId: "user-b",
    factoryId: "ws-payments",
    fundingSource: "byok",
    costCents: 80,
    totalTokens: 2000,
  }),
  event({ id: "old-model", occurredAt: "2026-07-01T12:00:00.000Z", costCents: 900, totalTokens: 20000 }),
];

describe("rangeForPreset", () => {
  it("uses rolling windows that end at now", () => {
    expect(rangeForPreset("day", NOW)).toEqual({
      start: new Date("2026-09-02T12:00:00.000Z"),
      end: NOW,
    });
    expect(rangeForPreset("week", NOW).start.toISOString()).toBe("2026-08-27T12:00:00.000Z");
    expect(rangeForPreset("month", NOW).start.toISOString()).toBe("2026-08-04T12:00:00.000Z");
    expect(rangeForPreset("year", NOW).start.toISOString()).toBe("2025-09-03T12:00:00.000Z");
  });
});

describe("rangeFromCustomDays", () => {
  it("keeps the last selected day inclusive", () => {
    const range = rangeFromCustomDays(new Date("2026-08-01T18:00:00.000Z"), new Date("2026-08-15T03:00:00.000Z"));
    expect(range.start.toISOString()).toBe("2026-08-01T00:00:00.000Z");
    expect(range.end.toISOString()).toBe("2026-08-16T00:00:00.000Z");
  });
});

describe("filterSpendingEvents", () => {
  it("keeps events in the range and drops older rows", () => {
    const matched = filterSpendingEvents(ledger, rangeForPreset("week", NOW), EMPTY_SPENDING_FILTERS);
    expect(matched.map((item) => item.id)).toEqual(["today-model", "today-compute", "week-gpt"]);
  });

  it("applies workspace and user filters together", () => {
    const matched = filterSpendingEvents(ledger, rangeForPreset("week", NOW), {
      ...EMPTY_SPENDING_FILTERS,
      workspaceIds: ["ws-refunds"],
      userIds: ["user-a"],
    });
    expect(matched.map((item) => item.id)).toEqual(["today-model", "today-compute"]);
  });

  it("limits model events without dropping unmatched compute", () => {
    const matched = filterSpendingEvents(ledger, rangeForPreset("week", NOW), {
      ...EMPTY_SPENDING_FILTERS,
      models: [modelKey("openai", "gpt-4o")],
    });
    expect(matched.map((item) => item.id)).toEqual(["today-compute", "week-gpt"]);
  });
});

describe("buildSpendingReport", () => {
  it("totals spend, tokens, and VM time for the month window", () => {
    const report = buildSpendingReport(
      ledger,
      rangeForPreset("month", NOW),
      EMPTY_SPENDING_FILTERS,
      "workspace",
      catalogs,
    );
    expect(report.totals).toEqual({
      costCents: 370,
      tokens: 6000,
      durationSeconds: 600,
      hostedCostCents: 290,
      byokCostCents: 80,
    });
    expect(report.breakdown.map((row) => row.id)).toEqual(["ws-refunds", "ws-payments"]);
    expect(report.breakdown[0]?.costCents).toBe(290);
  });

  it("groups model spend without mixing in VM rows", () => {
    const report = buildSpendingReport(ledger, rangeForPreset("week", NOW), EMPTY_SPENDING_FILTERS, "model", catalogs);
    expect(report.breakdown.map((row) => row.id)).toEqual(["anthropic/claude-sonnet-4-6", "openai/gpt-4o"]);
    expect(report.series.some((point) => (point.values["anthropic/claude-sonnet-4-6"] ?? 0) > 0)).toBe(true);
  });
});

describe("filter helpers", () => {
  it("toggles selected ids and reports an active filter set", () => {
    expect(toggleFilterValue(["a"], "b")).toEqual(["a", "b"]);
    expect(toggleFilterValue(["a", "b"], "a")).toEqual(["b"]);
    expect(hasActiveSpendingFilters(EMPTY_SPENDING_FILTERS)).toBe(false);
    expect(hasActiveSpendingFilters({ ...EMPTY_SPENDING_FILTERS, models: ["openai/gpt-4o"] })).toBe(true);
  });

  it("labels filter triggers and the visible date caption", () => {
    expect(formatFilterTriggerLabel("All users", 0, "user")).toBe("All users");
    expect(formatFilterTriggerLabel("All users", 1, "user")).toBe("1 user");
    expect(formatFilterTriggerLabel("All users", 2, "user")).toBe("2 users");
    expect(formatSpendingRangeCaption(rangeForPreset("day", NOW))).toBe("Sep 2, 2026 – Sep 3, 2026");
  });
});
