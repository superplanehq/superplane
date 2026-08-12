import { describe, expect, it } from "vitest";
import { formatFactoryNodeDuration, normalizeFactoryNodeStatus } from "./status";

describe("normalizeFactoryNodeStatus", () => {
  it("maps execution states onto factory card statuses", () => {
    expect(normalizeFactoryNodeStatus("success")).toBe("passed");
    expect(normalizeFactoryNodeStatus("failed")).toBe("failed");
    expect(normalizeFactoryNodeStatus("running")).toBe("running");
    expect(normalizeFactoryNodeStatus("queued")).toBe("pending");
    expect(normalizeFactoryNodeStatus(undefined)).toBe("pending");
  });
});

describe("formatFactoryNodeDuration", () => {
  it("formats compact durations like the Storybook factory cards", () => {
    expect(formatFactoryNodeDuration(7000)).toBe("7s");
    expect(formatFactoryNodeDuration(76_000)).toBe("1m 16s");
    expect(formatFactoryNodeDuration(9 * 60_000, { soFar: true })).toBe("9m so far");
  });
});
