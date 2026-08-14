import { describe, expect, it } from "vitest";
import { DEFAULT_EVENT_STATE_MAP } from "../componentBase/defaultEventStateMap";
import {
  factoryNodeStatusLabel,
  formatFactoryNodeDuration,
  normalizeFactoryNodeStatus,
  resolveFactoryRuntimeStatus,
} from "./status";

describe("normalizeFactoryNodeStatus", () => {
  it("maps classic event states onto factory card statuses", () => {
    expect(normalizeFactoryNodeStatus("success")).toBe("passed");
    expect(normalizeFactoryNodeStatus("failed")).toBe("failed");
    expect(normalizeFactoryNodeStatus("error")).toBe("error");
    expect(normalizeFactoryNodeStatus("running")).toBe("running");
    expect(normalizeFactoryNodeStatus("cancelling")).toBe("cancelling");
    expect(normalizeFactoryNodeStatus("queued")).toBe("queued");
    expect(normalizeFactoryNodeStatus("cancelled")).toBe("cancelled");
    expect(normalizeFactoryNodeStatus("triggered")).toBe("triggered");
    expect(normalizeFactoryNodeStatus("neutral")).toBe("pending");
    expect(normalizeFactoryNodeStatus(undefined)).toBe("pending");
  });

  it("maps If and loop channel outcomes so executed nodes are not Pending", () => {
    expect(normalizeFactoryNodeStatus("true")).toBe("passed");
    expect(normalizeFactoryNodeStatus("false")).toBe("passed");
    expect(normalizeFactoryNodeStatus("done")).toBe("passed");
    expect(normalizeFactoryNodeStatus("next")).toBe("running");
    expect(normalizeFactoryNodeStatus("waiting")).toBe("running");
    expect(normalizeFactoryNodeStatus("rejected")).toBe("cancelled");
  });

  it("maps mapper success aliases from the eventStateMap success style", () => {
    const stateMap = {
      ...DEFAULT_EVENT_STATE_MAP,
      "marked ready": DEFAULT_EVENT_STATE_MAP.success,
      merged: DEFAULT_EVENT_STATE_MAP.success,
    };

    expect(normalizeFactoryNodeStatus("marked ready", stateMap)).toBe("passed");
    expect(normalizeFactoryNodeStatus("merged", stateMap)).toBe("passed");
  });

  it("maps custom running styles instead of treating every alias as Passed", () => {
    const stateMap = {
      ...DEFAULT_EVENT_STATE_MAP,
      "is running": DEFAULT_EVENT_STATE_MAP.running,
    };

    expect(normalizeFactoryNodeStatus("is running", stateMap)).toBe("running");
  });

  it("maps custom cancelled styles such as GitHub workflow stopped", () => {
    const stateMap = {
      ...DEFAULT_EVENT_STATE_MAP,
      stopped: {
        icon: "circle-stop",
        textColor: "text-gray-800",
        backgroundColor: "bg-gray-100",
        badgeColor: "bg-gray-500",
      },
    };

    expect(normalizeFactoryNodeStatus("stopped", stateMap)).toBe("cancelled");
  });

  it("maps circle-stop icons even when the badge color is not a default", () => {
    const stateMap = {
      ...DEFAULT_EVENT_STATE_MAP,
      stopped: {
        icon: "circle-stop",
        textColor: "text-gray-800",
        backgroundColor: "bg-slate-100",
        badgeColor: "bg-slate-500",
      },
    };

    expect(normalizeFactoryNodeStatus("stopped", stateMap)).toBe("cancelled");
  });

  it("does not classify success icons by badge color alone", () => {
    const stateMap = {
      ...DEFAULT_EVENT_STATE_MAP,
      completed: {
        icon: "circle-check",
        textColor: "text-gray-800",
        backgroundColor: "bg-gray-100",
        badgeColor: "bg-gray-500",
      },
    };

    expect(normalizeFactoryNodeStatus("completed", stateMap)).toBe("passed");
  });

  it("treats unlisted mapper success aliases as Passed when no map is given", () => {
    expect(normalizeFactoryNodeStatus("marked ready")).toBe("passed");
  });
});

describe("resolveFactoryRuntimeStatus", () => {
  it("keeps Pending for unset state while the run is still active", () => {
    expect(resolveFactoryRuntimeStatus({ eventState: undefined, runIsActive: true })).toBe("pending");
    expect(resolveFactoryRuntimeStatus({ eventState: "neutral", runIsActive: true })).toBe("pending");
    expect(resolveFactoryRuntimeStatus({ eventState: "pending", runIsActive: true })).toBe("pending");
  });

  it("uses Did not run for unset state when the run has finished", () => {
    expect(resolveFactoryRuntimeStatus({ eventState: undefined, runIsActive: false })).toBe("did_not_run");
    expect(resolveFactoryRuntimeStatus({ eventState: "neutral", runIsActive: false })).toBe("did_not_run");
  });

  it("still maps channel and classic states when the run has finished", () => {
    expect(resolveFactoryRuntimeStatus({ eventState: "true", runIsActive: false })).toBe("passed");
    expect(resolveFactoryRuntimeStatus({ eventState: "success", runIsActive: false })).toBe("passed");
    expect(resolveFactoryRuntimeStatus({ eventState: "failed", runIsActive: false })).toBe("failed");
  });

  it("maps mapper success aliases when the run has finished", () => {
    const stateMap = {
      ...DEFAULT_EVENT_STATE_MAP,
      "marked ready": DEFAULT_EVENT_STATE_MAP.success,
    };

    expect(resolveFactoryRuntimeStatus({ eventState: "marked ready", runIsActive: false, stateMap })).toBe("passed");
  });

  it("defaults runIsActive to true when omitted", () => {
    expect(resolveFactoryRuntimeStatus({ eventState: undefined })).toBe("pending");
  });
});

describe("factoryNodeStatusLabel", () => {
  it("labels Did not run without a contraction", () => {
    expect(factoryNodeStatusLabel("did_not_run")).toBe("Did not run");
  });
});

describe("formatFactoryNodeDuration", () => {
  it("formats compact durations like the Storybook factory cards", () => {
    expect(formatFactoryNodeDuration(7000)).toBe("7s");
    expect(formatFactoryNodeDuration(76_000)).toBe("1m 16s");
    expect(formatFactoryNodeDuration(9 * 60_000, { soFar: true })).toBe("9m so far");
  });
});
