import { describe, expect, it } from "vitest";

import { formatWorkOrderDateTime } from "./workOrderDateTime";

describe("formatWorkOrderDateTime", () => {
  it("uses the compact 24-hour work-order format", () => {
    expect(formatWorkOrderDateTime(new Date(2026, 7, 6, 10, 17))).toBe("Aug 6 10:17");
  });

  it("returns an empty string for an invalid date", () => {
    expect(formatWorkOrderDateTime(new Date("invalid"))).toBe("");
  });
});
