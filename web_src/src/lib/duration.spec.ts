import { describe, expect, it } from "vitest";
import { formatDuration } from "@/lib/duration";

describe("duration", () => {
  it("formats milliseconds-only durations", () => {
    expect(formatDuration(250)).toBe("250ms");
  });

  it("formats second and millisecond durations", () => {
    expect(formatDuration(1_500)).toBe("1s 500ms");
  });

  it("formats minute and second durations", () => {
    expect(formatDuration(125_000)).toBe("2m 5s");
  });

  it("formats hour and minute durations", () => {
    expect(formatDuration(5_400_000)).toBe("1h 30m");
  });

  it("falls back to zero milliseconds for zero or negative durations", () => {
    expect(formatDuration(0)).toBe("");
    expect(formatDuration(-500)).toBe("");
  });
});
