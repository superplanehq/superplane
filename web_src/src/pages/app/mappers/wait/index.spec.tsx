import { act, cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import type { ExecutionInfo, SubtitleContext } from "../types";
import { waitMapper } from "./index";

const now = Math.floor(Date.now() / 1000) * 1000;
const createdAt = new Date(now - 10_000).toISOString();
const updatedAt = new Date(now - 5_000).toISOString();
const targetDate = new Date(now + 20_000).toISOString();
const interval = { mode: "interval", waitFor: "30", unit: "seconds" };
const legacyDuration = { value: 30, unit: "seconds" };

function subtitle(configuration: unknown = interval, overrides: Partial<ExecutionInfo> = {}) {
  const context: SubtitleContext = {
    node: {
      id: "wait-1",
      name: "Wait",
      componentName: "wait",
      isCollapsed: false,
      configuration,
    },
    execution: {
      id: "execution-1",
      createdAt,
      updatedAt,
      state: "STATE_STARTED",
      result: "RESULT_UNKNOWN",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: { id: "event-1", createdAt, data: {}, nodeId: "start-1", type: "trigger" },
      ...overrides,
    },
  };
  return waitMapper.subtitle(context);
}

function renderedSubtitle(configuration: unknown = interval, overrides: Partial<ExecutionInfo> = {}) {
  return render(<>{subtitle(configuration, overrides)}</>).container;
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(now);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("Wait subtitle duration resolution", () => {
  it.each([
    ["positive metadata", { interval_duration: 70_000 }, "Time left: 1m · 5s ago"],
    ["numeric string metadata", { interval_duration: "30000" }, "Time left: 20s · 5s ago"],
    ["missing metadata", undefined, "Time left: 20s · 5s ago"],
    ["missing metadata duration", {}, "Time left: 20s · 5s ago"],
    ["zero metadata", { interval_duration: 0 }, "Time left: 20s · 5s ago"],
    ["negative metadata", { interval_duration: -1 }, "Time left: 20s · 5s ago"],
    ["unusable metadata", { interval_duration: "invalid" }, "Time left: 20s · 5s ago"],
    ["NaN metadata", { interval_duration: NaN }, "Time left: 20s · 5s ago"],
    ["unusable metadata container", "invalid", "Time left: 20s · 5s ago"],
  ])("uses the current precedence for %s", (_name, metadata, expected) => {
    expect(renderedSubtitle(interval, { metadata }).textContent).toBe(expected);
  });

  it("falls back to configuration when metadata access fails", () => {
    const metadata = {
      get interval_duration(): number {
        throw new Error("Unavailable duration");
      },
    };
    expect(renderedSubtitle(interval, { metadata }).textContent).toBe("Time left: 20s · 5s ago");
  });

  it("does not read configuration when positive metadata supplies the duration", () => {
    const configuration = {
      get mode(): string {
        throw new Error("Unavailable configuration");
      },
    };
    expect(renderedSubtitle(configuration, { metadata: { interval_duration: 30_000 } }).textContent).toBe(
      "Time left: 20s · 5s ago",
    );
  });

  it.each([
    ["seconds", { ...interval, waitFor: "30" }, "Time left: 20s · 5s ago"],
    ["minutes", { ...interval, waitFor: "2", unit: "minutes" }, "Time left: 1m · 5s ago"],
    ["hours", { ...interval, waitFor: "2", unit: "hours" }, "Time left: 1h · 5s ago"],
    ["partial numeric input", { ...interval, waitFor: "30seconds" }, "Time left: 20s · 5s ago"],
    ["unknown unit", { ...interval, unit: "unknown" }, "Time left: 20s · 5s ago"],
    ["zero interval", { ...interval, waitFor: "0" }, "5s ago"],
    ["negative interval", { ...interval, waitFor: "-1" }, "Time left: 0s · 5s ago"],
    ["invalid interval with legacy duration", { ...interval, waitFor: "invalid", duration: legacyDuration }, "5s ago"],
    ["missing unit with legacy duration", { ...interval, unit: "", duration: legacyDuration }, "5s ago"],
    ["legacy seconds", { duration: legacyDuration }, "Time left: 20s · 5s ago"],
    ["legacy minutes", { duration: { value: 2, unit: "minutes" } }, "Time left: 1m · 5s ago"],
    ["legacy hours", { duration: { value: 2, unit: "hours" } }, "Time left: 1h · 5s ago"],
    ["legacy unknown unit", { duration: { value: 30, unit: "unknown" } }, "Time left: 20s · 5s ago"],
    ["legacy zero", { duration: { value: 0, unit: "seconds" } }, "5s ago"],
    ["legacy negative", { duration: { value: -1, unit: "seconds" } }, "Time left: 0s · 5s ago"],
    ["legacy invalid value", { duration: { value: "invalid", unit: "seconds" } }, "5s ago"],
    [
      "unrecognized mode with legacy duration",
      { mode: "unknown", duration: legacyDuration },
      "Time left: 20s · 5s ago",
    ],
    ["missing configuration", null, "5s ago"],
  ])("preserves %s configuration behavior", (_name, configuration, expected) => {
    expect(renderedSubtitle(configuration).textContent).toBe(expected);
  });

  it.each([
    ["double-quoted target", `date("${targetDate}")`, "Time left: 20s · 5s ago"],
    ["single-quoted target", `date('${targetDate}')`, "Time left: 20s · 5s ago"],
    ["unquoted target", targetDate, "5s ago"],
    ["invalid date", 'date("invalid")', "5s ago"],
    ["missing target", undefined, "5s ago"],
    ["zero duration", `date("${createdAt}")`, "5s ago"],
    ["negative duration", `date("${new Date(now - 20_000).toISOString()}")`, "Time left: 0s · 5s ago"],
  ])("preserves countdown behavior for %s", (_name, waitUntil, expected) => {
    expect(renderedSubtitle({ mode: "countdown", waitUntil, duration: legacyDuration }).textContent).toBe(expected);
  });

  it.each([
    ["missing creation timestamp", "", "5s ago"],
    ["invalid creation timestamp", "invalid", "5s ago"],
  ])("uses relative time alone for a countdown with %s", (_name, timestamp, expected) => {
    expect(
      renderedSubtitle({ mode: "countdown", waitUntil: `date("${targetDate}")` }, { createdAt: timestamp }).textContent,
    ).toBe(expected);
  });

  it("updates the running countdown and relative timestamp", () => {
    const container = renderedSubtitle();
    expect(container.textContent).toBe("Time left: 20s · 5s ago");
    act(() => vi.advanceTimersByTime(1000));
    expect(container.textContent).toBe("Time left: 19s · 6s ago");
  });

  it("uses creation time when the update timestamp is missing", () => {
    expect(renderedSubtitle(interval, { updatedAt: "" }).textContent).toBe("Time left: 20s · 10s ago");
  });

  it("uses relative time alone when a running execution has no creation timestamp", () => {
    expect(renderedSubtitle(interval, { createdAt: "" }).textContent).toBe("5s ago");
  });

  it("returns the public empty-string fallback when both timestamps are missing", () => {
    expect(subtitle(interval, { createdAt: "", updatedAt: "" })).toBe("");
  });
});

describe("Wait terminal subtitles", () => {
  const terminalExecutions: [string, Partial<ExecutionInfo>][] = [
    ["finished", { state: "STATE_FINISHED", result: "RESULT_PASSED" }],
    ["failed", { state: "STATE_FINISHED", result: "RESULT_FAILED", resultMessage: "" }],
    [
      "pushed through",
      {
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        outputs: { default: [{ data: { reason: "manual_override" } }] },
      },
    ],
  ];

  it.each(terminalExecutions)("shows the completion timestamp for %s", (_name, execution) => {
    const completionTime = new Date(updatedAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    expect(renderedSubtitle(interval, execution).textContent).toBe(`Done at: ${completionTime} · 5s ago`);
  });

  it.each(terminalExecutions)("falls back to creation time for %s", (_name, execution) => {
    expect(renderedSubtitle(interval, { ...execution, updatedAt: "" }).textContent).toBe("Done · 10s ago");
  });

  it.each(terminalExecutions)("shows Done without timestamps for %s", (_name, execution) => {
    expect(renderedSubtitle(interval, { ...execution, createdAt: "", updatedAt: "" }).textContent).toBe("Done");
  });
});
