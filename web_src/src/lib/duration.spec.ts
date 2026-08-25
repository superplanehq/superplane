import { afterEach, describe, expect, it, vi } from "vitest";
import { formatDuration, formatMinutesSecondsDuration } from "@/lib/duration";

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

  it("never hands a non-finite duration to Intl.DurationFormat", () => {
    const format = vi.fn((duration: Record<string, number>) => {
      // Matches the runtime: Intl.DurationFormat rejects non-finite values.
      for (const value of Object.values(duration)) {
        if (!Number.isFinite(value)) throw new RangeError("Expected finite integer");
      }
      return "formatted-by-intl";
    });
    const DurationFormat = vi.fn().mockImplementation(function () {
      return { format };
    });

    vi.stubGlobal("Intl", {
      ...Intl,
      DurationFormat,
    });

    expect(formatDuration(Number.NaN)).toBe("");
    expect(formatDuration(Number.POSITIVE_INFINITY)).toBe("");
    expect(format).not.toHaveBeenCalled();
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

  it("returns an empty string for non-finite minute and second durations", () => {
    expect(formatMinutesSecondsDuration(Number.NaN)).toBe("");
    expect(formatMinutesSecondsDuration(Number.POSITIVE_INFINITY)).toBe("");
    expect(formatMinutesSecondsDuration(Number.NEGATIVE_INFINITY)).toBe("");
  });

  it("formats minutes and seconds without milliseconds", () => {
    expect(formatMinutesSecondsDuration(0)).toBe("");
    expect(formatMinutesSecondsDuration(250)).toBe("<1s");
    expect(formatMinutesSecondsDuration(1_500)).toBe("1s");
    expect(formatMinutesSecondsDuration(51_988)).toBe("51s");
    expect(formatMinutesSecondsDuration(61_500)).toBe("1m 1s");
    expect(formatMinutesSecondsDuration(5_400_000)).toBe("90m");
  });
});

describe("formatDuration with { precision: 'second' }", () => {
  const seconds = (durationMs: number) => formatDuration(durationMs, { precision: "second" });

  it("returns an empty string for zero, negative, or non-finite durations", () => {
    expect(seconds(0)).toBe("");
    expect(seconds(-500)).toBe("");
    expect(seconds(NaN)).toBe("");
    expect(seconds(Infinity)).toBe("");
  });

  it("renders sub-second durations as '< 1s'", () => {
    expect(seconds(1)).toBe("< 1s");
    expect(seconds(482)).toBe("< 1s");
    expect(seconds(999)).toBe("< 1s");
  });

  it("rounds to the nearest whole second and never renders milliseconds", () => {
    expect(seconds(1_000)).toBe("1s");
    expect(seconds(1_499)).toBe("1s");
    expect(seconds(61_500)).toBe("1m 2s");
    expect(seconds(16_482)).toBe("16s");
  });

  it("formats minute and hour durations without milliseconds", () => {
    expect(seconds(125_000)).toBe("2m 5s");
    expect(seconds(5_400_000)).toBe("1h 30m");
  });

  it("rolls over into days for multi-day durations", () => {
    expect(seconds(90_000_000)).toBe("1d 1h");
  });
});
