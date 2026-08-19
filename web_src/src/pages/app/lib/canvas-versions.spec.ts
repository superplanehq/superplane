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

    expect(formatVersionTimestamp({ updatedAt })).toBe(
      new Date(updatedAt).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }),
    );
  });

  it("returns undefined for invalid timestamps", () => {
    expect(formatVersionTimestamp({ updatedAt: "not-a-date" })).toBeUndefined();
  });

  it("returns undefined when no timestamp is present", () => {
    expect(formatVersionTimestamp({})).toBeUndefined();
  });
});

describe("formatVersionLabel", () => {
  it("uses the commit message when present", () => {
    expect(formatVersionLabel({ commitMessage: "Fix webhook retry" })).toBe("Fix webhook retry");
  });

  it("falls back to a timestamp-based label when commit message is missing", () => {
    const createdAt = "2026-05-18T12:00:00.000Z";
    const expectedTimestamp = new Date(createdAt).toLocaleString(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    });

    expect(formatVersionLabel({ createdAt })).toBe(`Update from ${expectedTimestamp}`);
  });

  it("falls back to an untitled label when commit message and createdAt are missing", () => {
    expect(formatVersionLabel({})).toBe("Untitled update");
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
        commitMessage: "Initial setup",
        createdAt,
      }),
    ).toBe(`Initial setup · ${expectedTimestamp}`);
  });

  it("returns only the label when no valid timestamp exists", () => {
    expect(formatVersionLabelWithTimestamp({ commitMessage: "Initial setup" })).toBe("Initial setup");
  });

  it("uses a legacy timestamp label without duplicating the timestamp suffix", () => {
    const createdAt = "2026-05-18T12:00:00.000Z";
    const expectedTimestamp = new Date(createdAt).toLocaleString(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    });

    expect(formatVersionLabelWithTimestamp({ createdAt })).toBe(`Update from ${expectedTimestamp}`);
  });
});

describe("sortVersionsDesc", () => {
  it("sorts versions by updatedAt descending", () => {
    const sorted = sortVersionsDesc([
      {
        id: "older",
        updatedAt: "2026-06-01T12:00:00.000Z",
      },
      {
        id: "newer",
        updatedAt: "2026-06-03T12:00:00.000Z",
      },
    ]);

    expect(sorted.map((version) => version.id)).toEqual(["newer", "older"]);
  });

  it("falls back to createdAt when updatedAt is missing", () => {
    const sorted = sortVersionsDesc([
      {
        id: "bbb",
        createdAt: "2026-06-01T12:00:00.000Z",
      },
      {
        id: "aaa",
        createdAt: "2026-06-03T12:00:00.000Z",
      },
    ]);

    expect(sorted.map((version) => version.id)).toEqual(["aaa", "bbb"]);
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
