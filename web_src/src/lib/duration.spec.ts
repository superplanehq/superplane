import { afterEach, describe, expect, it, vi } from "vitest";
import { formatDuration } from "@/lib/duration";

describe("duration", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("uses Intl.DurationFormat when available", () => {
    const format = vi.fn().mockReturnValue("formatted-by-intl");
    const DurationFormat = vi.fn().mockImplementation(function () {
      return { format };
    });

    vi.stubGlobal("Intl", {
      ...Intl,
      DurationFormat,
    });

    expect(formatDuration(1_500)).toBe("formatted-by-intl");
    expect(DurationFormat).toHaveBeenCalledWith(undefined, { style: "narrow" });
    expect(format).toHaveBeenCalledWith({ seconds: 1, milliseconds: 500 });
  });

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

  it("falls back when Intl.DurationFormat is unavailable", () => {
    vi.stubGlobal("Intl", {
      ...Intl,
      DurationFormat: undefined,
    });

    expect(formatDuration(250)).toBe("250ms");
    expect(formatDuration(1_500)).toBe("1s 500ms");
    expect(formatDuration(125_000)).toBe("2m 5s");
    expect(formatDuration(5_400_000)).toBe("1h 30m");
    expect(formatDuration(0)).toBe("");
  });
});
