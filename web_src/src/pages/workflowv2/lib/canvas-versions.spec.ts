import { describe, expect, it } from "vitest";
import {
  formatVersionLabel,
  formatVersionLabelWithTimestamp,
  formatVersionTimestamp,
  versionSortValue,
} from "./canvas-versions";

describe("formatVersionTimestamp", () => {
  it("formats the first available timestamp", () => {
    const updatedAt = "2026-03-29T12:34:56.000Z";

    expect(
      formatVersionTimestamp({
        metadata: {
          updatedAt,
        },
      }),
    ).toBe(new Date(updatedAt).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }));
  });

  it("returns undefined for invalid timestamps", () => {
    expect(
      formatVersionTimestamp({
        metadata: {
          updatedAt: "not-a-date",
        },
      }),
    ).toBeUndefined();
  });

  it("returns undefined when no timestamp is present", () => {
    expect(formatVersionTimestamp({ metadata: {} })).toBeUndefined();
  });
});

describe("formatVersionLabel", () => {
  it("labels published versions", () => {
    expect(formatVersionLabel({ metadata: { isPublished: true } })).toBe("Published version");
  });

  it("labels unpublished versions as drafts", () => {
    expect(formatVersionLabel({ metadata: { isPublished: false } })).toBe("Draft version");
  });
});

describe("formatVersionLabelWithTimestamp", () => {
  it("appends the formatted timestamp when present", () => {
    const createdAt = "2026-03-29T12:34:56.000Z";
    const expectedTimestamp = new Date(createdAt).toLocaleString(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    });

    expect(
      formatVersionLabelWithTimestamp({
        metadata: {
          isPublished: false,
          createdAt,
        },
      }),
    ).toBe(`Draft version · ${expectedTimestamp}`);
  });

  it("returns only the label when no valid timestamp exists", () => {
    expect(formatVersionLabelWithTimestamp({ metadata: { isPublished: true } })).toBe("Published version");
  });
});

describe("versionSortValue", () => {
  it("returns the parsed timestamp value", () => {
    expect(versionSortValue("2026-03-29T12:34:56.000Z")).toBe(Date.parse("2026-03-29T12:34:56.000Z"));
  });

  it("returns zero for missing or invalid timestamps", () => {
    expect(versionSortValue()).toBe(0);
    expect(versionSortValue("not-a-date")).toBe(0);
  });
});
