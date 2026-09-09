import { describe, expect, it } from "vitest";

import {
  firstPositiveWorkOrderMetric,
  formatDurationSeconds,
  formatUsageOccurredAtUtc,
  formatUsageRunResources,
  formatUsageTaskName,
  formatUsageTokensAndTime,
  formatWorkOrderExecutionUsage,
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

describe("formatUsageRunResources", () => {
  it("notes your keys on the model when BYOK paid the model", () => {
    expect(formatUsageRunResources(["anthropic/claude-sonnet-4-6"], ["e1-large-amd64"], true)).toBe(
      "anthropic/claude-sonnet-4-6 (your keys) · e1-large-amd64",
    );
  });

  it("keeps hosted models without the your-keys note", () => {
    expect(formatUsageRunResources(["anthropic/claude-sonnet-4-6"], ["e1-large-amd64"], false)).toBe(
      "anthropic/claude-sonnet-4-6 · e1-large-amd64",
    );
  });

  it("shows only the parts that exist", () => {
    expect(formatUsageRunResources(["anthropic/claude-sonnet-4-6"], [], false)).toBe("anthropic/claude-sonnet-4-6");
    expect(formatUsageRunResources([], ["e1-large-amd64"], false)).toBe("e1-large-amd64");
    expect(formatUsageRunResources([], [], false)).toBe("—");
  });
});

describe("formatUsageOccurredAtUtc", () => {
  it("formats an ISO timestamp in UTC", () => {
    expect(formatUsageOccurredAtUtc("2026-09-08T15:04:00Z")).toBe("Sep 8, 2026 15:04 UTC");
  });

  it("returns an em dash for missing or invalid values", () => {
    expect(formatUsageOccurredAtUtc(undefined)).toBe("—");
    expect(formatUsageOccurredAtUtc("not-a-date")).toBe("—");
  });
});
