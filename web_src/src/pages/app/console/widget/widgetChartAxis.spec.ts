import { describe, expect, it } from "vitest";

import { formatXAxisTick, formatXTooltipLabel } from "./widgetChartAxis";

describe("widgetChartAxis timestamp auto-detect", () => {
  it("keeps categorical numeric strings as labels when xFormat is unset", () => {
    for (const category of ["12", "200", "404", "1", "99"]) {
      expect(formatXAxisTick(category, undefined)).toBe(category);
      expect(formatXTooltipLabel(category, undefined)).toBe(category);
    }
  });

  it("still formats ISO timestamps as compact dates without xFormat", () => {
    expect(formatXAxisTick("2026-05-26T00:00:00Z", undefined)).toMatch(/May/);
    expect(formatXAxisTick("2026-05-26T00:00:00Z", undefined)).not.toMatch(/T00:00:00/);
  });

  it("still formats plausible epoch numeric strings as dates without xFormat", () => {
    const seconds = Math.floor(Date.parse("2026-05-26T00:00:00Z") / 1000);
    expect(formatXAxisTick(String(seconds), undefined)).toMatch(/May/);
  });

  it("honors explicit date/datetime xFormat even for numeric epoch values", () => {
    const seconds = Math.floor(Date.parse("2026-05-26T16:10:00Z") / 1000);
    expect(formatXAxisTick(seconds, "date")).toMatch(/May/);
    expect(formatXTooltipLabel(seconds, "datetime")).toMatch(/May/);
    expect(formatXTooltipLabel(seconds, "datetime")).toMatch(/\d/);
  });

  it("keeps categorical digit strings as labels even with explicit date xFormat", () => {
    expect(formatXAxisTick("404", "date")).toBe("404");
    expect(formatXTooltipLabel("404", "datetime")).toBe("404");
    expect(formatXAxisTick(12, "date")).toBe("12");
  });

  it("uses compact date axis ticks for xFormat relative, not live relative text", () => {
    const iso = "2026-05-26T16:10:00Z";
    expect(formatXAxisTick(iso, "relative")).toMatch(/May/);
    expect(formatXAxisTick(iso, "relative")).not.toMatch(/ago|in \d/);
    expect(formatXTooltipLabel(iso, "relative")).toMatch(/May/);
  });
});
