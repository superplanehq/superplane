import { describe, expect, it } from "vitest";

import { formatSubHourDuration, subHourVelocityDurationFormat } from "./velocityDurationFormat";

describe("formatSubHourDuration", () => {
  it("formats sub-hour values as minutes", () => {
    expect(formatSubHourDuration(0.2)).toBe("12m");
  });

  it("formats in-range values as whole hours", () => {
    expect(formatSubHourDuration(5)).toBe("5h");
  });

  it("formats large values as days", () => {
    expect(formatSubHourDuration(60)).toBe("2.5d");
  });

  it("still reads a genuine zero as 0h", () => {
    expect(formatSubHourDuration(0)).toBe("0h");
  });
});

describe("subHourVelocityDurationFormat.pickChartUnit", () => {
  it("picks minutes when every value in the period is sub-hour", () => {
    const unit = subHourVelocityDurationFormat.pickChartUnit([0.1, 0.2, 0.35]);
    expect(unit.unit).toBe("m");
    expect(unit.formatTick(0.2)).toBe("12m");
  });

  it("picks hours for a mixed period whose largest value is in range", () => {
    const unit = subHourVelocityDurationFormat.pickChartUnit([0, 0.2, 28]);
    expect(unit.unit).toBe("h");
    expect(unit.formatTick(28)).toBe("28h");
  });

  it("picks days when the largest value in the period is multi-day", () => {
    const unit = subHourVelocityDurationFormat.pickChartUnit([12, 60, 100]);
    expect(unit.unit).toBe("d");
    expect(unit.formatTick(120)).toBe("5d");
  });
});
