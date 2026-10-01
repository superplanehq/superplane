import { describe, expect, it } from "bun:test";
import { formatIntakeCreatedTime } from "./backlogIntakeCreatedTime";

describe("formatIntakeCreatedTime", () => {
  const now = new Date(2026, 9, 1, 12).getTime();
  const day = 24 * 60 * 60 * 1000;

  it("uses relative text before seven days and a calendar date at the cutoff", () => {
    expect(formatIntakeCreatedTime(now - 5 * 60 * 1000, "en-GB", now)).toBe("5 minutes ago");
    expect(formatIntakeCreatedTime(now - 2 * day, "en-GB", now)).toBe("2 days ago");
    expect(formatIntakeCreatedTime(now - 7 * day + 1, "en-GB", now)).toBe("1 week ago");
    expect(formatIntakeCreatedTime(now - 7 * day, "en-GB", now)).toBe("24 Sept 2026");
    expect(formatIntakeCreatedTime(now - 30 * day, "en-GB", now)).toBe("01 Sept 2026");
  });

  it("omits missing and invalid times", () => {
    expect(formatIntakeCreatedTime(undefined)).toBe("");
    expect(formatIntakeCreatedTime("")).toBe("");
    expect(formatIntakeCreatedTime("invalid")).toBe("");
  });
});
