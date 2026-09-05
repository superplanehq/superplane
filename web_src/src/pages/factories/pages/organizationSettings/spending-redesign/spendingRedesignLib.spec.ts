import { describe, expect, it } from "vitest";

import {
  buildSpendingReport,
  EMPTY_SPENDING_FILTERS,
  filterSpendingEvents,
  formatFilterTriggerLabel,
  formatSpendingRangeCaption,
  hasActiveSpendingFilters,
  MACHINE_BREAKDOWN_OPTIONS,
  MODEL_BREAKDOWN_OPTIONS,
  modelKey,
  rangeForPreset,
  rangeFromCustomDays,
  spendingPeriodTriggerLabel,
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

  it("limits one section to a single usage kind", () => {
    const models = filterSpendingEvents(ledger, rangeForPreset("week", NOW), EMPTY_SPENDING_FILTERS, "model");
    const machines = filterSpendingEvents(ledger, rangeForPreset("week", NOW), EMPTY_SPENDING_FILTERS, "compute");
    expect(models.map((item) => item.id)).toEqual(["today-model", "week-gpt"]);
    expect(machines.map((item) => item.id)).toEqual(["today-compute"]);
  });

  it("applies a single workspace and user together", () => {
    const matched = filterSpendingEvents(ledger, rangeForPreset("week", NOW), {
      ...EMPTY_SPENDING_FILTERS,
      workspaceId: "ws-refunds",
      userId: "user-a",
    });
    expect(matched.map((item) => item.id)).toEqual(["today-model", "today-compute"]);
  });

  it("keeps one selected model and drops other model rows", () => {
    const matched = filterSpendingEvents(
      ledger,
      rangeForPreset("week", NOW),
      { ...EMPTY_SPENDING_FILTERS, model: modelKey("openai", "gpt-4o") },
      "model",
    );
    expect(matched.map((item) => item.id)).toEqual(["week-gpt"]);
  });
});

describe("buildSpendingReport", () => {
  it("totals model spend without VM rows", () => {
    const report = buildSpendingReport({
      events: ledger,
      range: rangeForPreset("month", NOW),
      filters: EMPTY_SPENDING_FILTERS,
      breakdown: "workspace",
      catalogs,
      usageKind: "model",
    });
    expect(report.totals.costCents).toBe(330);
    expect(report.breakdown.map((row) => row.id)).toEqual(["ws-refunds", "ws-payments"]);
    expect(report.breakdown[0]?.costCents).toBe(250);
  });

  it("totals VM spend without model rows", () => {
    const report = buildSpendingReport({
      events: ledger,
      range: rangeForPreset("week", NOW),
      filters: EMPTY_SPENDING_FILTERS,
      breakdown: "machine",
      catalogs,
      usageKind: "compute",
    });
    expect(report.totals.costCents).toBe(40);
    expect(report.breakdown.map((row) => row.id)).toEqual(["e1-large-amd64"]);
    expect(report.series.some((point) => (point.values["e1-large-amd64"] ?? 0) > 0)).toBe(true);
  });

  it("groups model spend without mixing in VM rows", () => {
    const report = buildSpendingReport({
      events: ledger,
      range: rangeForPreset("week", NOW),
      filters: EMPTY_SPENDING_FILTERS,
      breakdown: "model",
      catalogs,
      usageKind: "model",
    });
    expect(report.breakdown.map((row) => row.id)).toEqual(["anthropic/claude-sonnet-4-6", "openai/gpt-4o"]);
    expect(report.breakdown.map((row) => row.label)).toEqual(["claude-sonnet-4-6", "gpt-4o"]);
    expect(report.seriesKeys.map((item) => item.label)).toEqual(["claude-sonnet-4-6", "gpt-4o"]);
    expect(report.series.some((point) => (point.values["anthropic/claude-sonnet-4-6"] ?? 0) > 0)).toBe(true);
  });

  it("labels stored family aliases from the catalog", () => {
    const report = buildSpendingReport({
      events: [event({ id: "alias-sonnet", occurredAt: "2026-09-03T10:00:00.000Z", model: "sonnet", costCents: 100 })],
      range: rangeForPreset("week", NOW),
      filters: EMPTY_SPENDING_FILTERS,
      breakdown: "model",
      catalogs: {
        ...catalogs,
        models: [{ id: "anthropic/sonnet", label: "claude-sonnet-4-6" }],
      },
      usageKind: "model",
    });
    expect(report.breakdown[0]).toMatchObject({ id: "anthropic/sonnet", label: "claude-sonnet-4-6" });
    expect(report.seriesKeys[0].label).toBe("claude-sonnet-4-6");
  });
});

describe("filter helpers", () => {
  it("reports an active single-select filter set", () => {
    expect(hasActiveSpendingFilters(EMPTY_SPENDING_FILTERS)).toBe(false);
    expect(hasActiveSpendingFilters({ ...EMPTY_SPENDING_FILTERS, model: "openai/gpt-4o" })).toBe(true);
  });

  it("labels filter triggers with the selected name", () => {
    expect(formatFilterTriggerLabel("All users")).toBe("All users");
    expect(formatFilterTriggerLabel("All users", "Alex")).toBe("Alex");
    expect(formatSpendingRangeCaption(rangeForPreset("day", NOW))).toBe("Sep 2, 2026 – Sep 3, 2026");
  });

  it("labels the period trigger with the preset or the custom dates", () => {
    expect(spendingPeriodTriggerLabel("month", rangeForPreset("month", NOW))).toBe("Last 30 days");
    expect(spendingPeriodTriggerLabel("custom", rangeForPreset("week", NOW))).toBe("Aug 27, 2026 – Sep 3, 2026");
  });

  it("exposes group-by options for each usage section", () => {
    expect(MODEL_BREAKDOWN_OPTIONS.map((option) => option.value)).toEqual(["workspace", "user", "model"]);
    expect(MACHINE_BREAKDOWN_OPTIONS.map((option) => option.value)).toEqual(["workspace", "user", "machine"]);
  });
});
