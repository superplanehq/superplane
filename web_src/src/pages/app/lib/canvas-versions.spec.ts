import { describe, expect, it } from "vitest";
import {
  formatVersionLabel,
  formatVersionLabelWithTimestamp,
  formatVersionTimestamp,
  sortVersionsDesc,
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
  it("uses the commit message when present", () => {
    expect(formatVersionLabel({ metadata: { commitMessage: "Fix webhook retry" } })).toBe("Fix webhook retry");
  });

  it("falls back to a generic label when commit message is missing", () => {
    expect(formatVersionLabel({ metadata: {} })).toBe("Commit");
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
          commitMessage: "Initial setup",
          createdAt,
        },
      }),
    ).toBe(`Initial setup · ${expectedTimestamp}`);
  });

  it("returns only the label when no valid timestamp exists", () => {
    expect(formatVersionLabelWithTimestamp({ metadata: { commitMessage: "Initial setup" } })).toBe("Initial setup");
  });
});

describe("sortVersionsDesc", () => {
  it("sorts versions by updatedAt descending", () => {
    const sorted = sortVersionsDesc([
      {
        metadata: {
          id: "older",
          updatedAt: "2026-06-01T12:00:00.000Z",
        },
      },
      {
        metadata: {
          id: "newer",
          updatedAt: "2026-06-03T12:00:00.000Z",
        },
      },
    ]);

    expect(sorted.map((version) => version.metadata?.id)).toEqual(["newer", "older"]);
  });

  it("falls back to createdAt when updatedAt is missing", () => {
    const sorted = sortVersionsDesc([
      {
        metadata: {
          id: "bbb",
          createdAt: "2026-06-01T12:00:00.000Z",
        },
      },
      {
        metadata: {
          id: "aaa",
          createdAt: "2026-06-03T12:00:00.000Z",
        },
      },
    ]);

    expect(sorted.map((version) => version.metadata?.id)).toEqual(["aaa", "bbb"]);
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
