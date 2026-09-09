import { describe, expect, it } from "vitest";

import {
  firstPositiveWorkOrderMetric,
  formatDurationSeconds,
  formatUsageMachineTypes,
  formatUsageModels,
  formatUsageOccurredAt,
  formatUsageSpend,
  formatUsageTaskKey,
  formatUsageTaskName,
  formatUsageTokensAndTime,
  formatWorkOrderExecutionUsage,
  usageTokenSpendCents,
  usageVmSpendCents,
} from "./workOrderUsage";

describe("firstPositiveWorkOrderMetric", () => {
  it("skips zero and empty values", () => {
    expect(firstPositiveWorkOrderMetric("0", undefined, "45")).toBe("45");
    expect(firstPositiveWorkOrderMetric(0, "1200")).toBe("1200");
    expect(firstPositiveWorkOrderMetric("0", "0")).toBeUndefined();
  });
});

describe("formatWorkOrderExecutionUsage", () => {
  it("sums usage from every execution in the dispatch", () => {
    expect(
      formatWorkOrderExecutionUsage([
        { totalTokens: "1200", costCents: "45", durationSeconds: "12" },
        { totalTokens: "1800", costCents: "105", durationSeconds: "8" },
      ]),
    ).toBe("$1.50 · 3k tokens · 20 s");
  });

  it("ignores absent and invalid usage", () => {
    expect(formatWorkOrderExecutionUsage([{ totalTokens: "invalid" }, {}])).toBeNull();
  });
});

describe("formatDurationSeconds", () => {
  it("shows seconds only under a minute", () => {
    expect(formatDurationSeconds(45)).toBe("45 s");
  });

  it("shows minutes at exactly 60 seconds", () => {
    expect(formatDurationSeconds(60)).toBe("1 min");
  });

  it("shows minutes and seconds remainder", () => {
    expect(formatDurationSeconds(90)).toBe("1 min 30 s");
  });

  it("shows hours at exactly 3600 seconds", () => {
    expect(formatDurationSeconds(3600)).toBe("1 h");
  });

  it("shows hours and minutes remainder", () => {
    expect(formatDurationSeconds(3900)).toBe("1 h 5 min");
    expect(formatDurationSeconds(5400)).toBe("1 h 30 min");
  });

  it("keeps hours growing for large values", () => {
    expect(formatDurationSeconds(7200)).toBe("2 h");
  });
});

describe("formatUsageTaskName", () => {
  it("joins the work-order key and title like the board", () => {
    expect(formatUsageTaskName("RF-101", "Publish draft")).toBe("RF-101 · Publish draft");
  });

  it("falls back to Untitled task when the title is empty", () => {
    expect(formatUsageTaskName("RF-101", "  ")).toBe("RF-101 · Untitled task");
    expect(formatUsageTaskName("", "")).toBe("Untitled task");
  });
});

describe("formatUsageTaskKey", () => {
  it("returns the work-order key without the title", () => {
    expect(formatUsageTaskKey("RF-101")).toBe("RF-101");
  });

  it("falls back to Untitled task when the key is empty", () => {
    expect(formatUsageTaskKey("  ")).toBe("Untitled task");
    expect(formatUsageTaskKey(undefined)).toBe("Untitled task");
  });
});

describe("formatUsageTokensAndTime", () => {
  it("joins tokens and VM time with the board separator", () => {
    expect(formatUsageTokensAndTime(22000, 90)).toBe("22k tokens · 1 min 30 s");
  });

  it("omits zero metrics", () => {
    expect(formatUsageTokensAndTime(22000, 0)).toBe("22k tokens");
    expect(formatUsageTokensAndTime(0, 45)).toBe("45 s");
    expect(formatUsageTokensAndTime(0, 0)).toBe("—");
  });
});

describe("formatUsageModels", () => {
  it("notes your keys on the model when your keys paid the model", () => {
    expect(formatUsageModels(undefined, ["anthropic/claude-sonnet-4-6"])).toBe(
      "anthropic/claude-sonnet-4-6 (your keys)",
    );
  });

  it("keeps hosted models without the your-keys note", () => {
    expect(formatUsageModels(["anthropic/claude-sonnet-4-6"])).toBe("anthropic/claude-sonnet-4-6");
  });

  it("notes your keys on only the your-keys models of a mixed run", () => {
    expect(formatUsageModels(["anthropic/claude-sonnet-4-6"], ["openai/gpt-5"])).toBe(
      "anthropic/claude-sonnet-4-6 · openai/gpt-5 (your keys)",
    );
  });

  it("returns an em dash when no models exist", () => {
    expect(formatUsageModels()).toBe("—");
  });
});

describe("formatUsageMachineTypes", () => {
  it("joins unique machine types", () => {
    expect(formatUsageMachineTypes(["e1-large-amd64", "e1-large-amd64"])).toBe("e1-large-amd64");
    expect(formatUsageMachineTypes(["e1-large-amd64", "e1-standard-amd64"])).toBe("e1-large-amd64 · e1-standard-amd64");
  });

  it("returns an em dash when no machine types exist", () => {
    expect(formatUsageMachineTypes()).toBe("—");
    expect(formatUsageMachineTypes([])).toBe("—");
  });
});

describe("formatUsageSpend", () => {
  it("formats a positive amount in USD", () => {
    expect(formatUsageSpend(123)).toBe("$1.23");
  });

  it("returns an em dash when there is no spend", () => {
    expect(formatUsageSpend(0)).toBe("—");
  });
});

describe("usageTokenSpendCents", () => {
  it("sums hosted and your-keys model spend", () => {
    expect(usageTokenSpendCents(120, 3)).toBe(123);
  });
});

describe("usageVmSpendCents", () => {
  it("returns the remainder after model spend", () => {
    expect(usageVmSpendCents(175, 150, 0)).toBe(25);
    expect(usageVmSpendCents(53, 0, 3)).toBe(50);
  });

  it("returns zero when model spend covers the total", () => {
    expect(usageVmSpendCents(123, 120, 3)).toBe(0);
  });
});

describe("formatUsageOccurredAt", () => {
  it("formats an ISO timestamp without the year or UTC suffix", () => {
    expect(formatUsageOccurredAt("2026-09-08T15:04:00Z")).toBe("Sep 8, 3:04 PM");
  });

  it("uses 12-hour time at morning hours", () => {
    expect(formatUsageOccurredAt("2026-09-09T07:57:00Z")).toBe("Sep 9, 7:57 AM");
  });

  it("returns an em dash for missing or invalid values", () => {
    expect(formatUsageOccurredAt(undefined)).toBe("—");
    expect(formatUsageOccurredAt("not-a-date")).toBe("—");
  });
});
