import { formatCentsPerMillion, formatMicrosPerHour } from "@/lib/priceBookRates";

import { describe, expect, it } from "vitest";

describe("priceBookRates", () => {
  it("formats model and VM prices", () => {
    expect(formatCentsPerMillion(300)).toBe("$3.00 / MTok");
    expect(formatMicrosPerHour(556)).toBe("$2.00 / hour");
  });
});
