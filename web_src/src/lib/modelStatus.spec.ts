import { describe, expect, it } from "bun:test";

import {
  MODEL_STATUS_DEFAULT_WINDOW,
  modelStatusBand,
  modelStatusCallCountLabel,
  modelStatusMatrixRows,
} from "./modelStatus";

describe("modelStatusBand", () => {
  it("maps scores to availability bands and leaves empty cells without a score", () => {
    expect(modelStatusBand(undefined)).toBe("no-data");
    expect(modelStatusBand(100)).toBe("available");
    expect(modelStatusBand(80)).toBe("available");
    expect(modelStatusBand(79)).toBe("degraded");
    expect(modelStatusBand(50)).toBe("degraded");
    expect(modelStatusBand(49)).toBe("unavailable");
    expect(modelStatusBand(0)).toBe("unavailable");
  });
});

describe("modelStatusMatrixRows", () => {
  it("sorts models by the lowest provider score and keeps empty cells as no data", () => {
    const rows = modelStatusMatrixRows([
      { provider: "anthropic", model: "claude-opus-4-6" },
      { provider: "anthropic", model: "claude-sonnet-4-6", score: 100, callCount: 3 },
      { provider: "openrouter", model: "claude-sonnet-4-6", score: 50, callCount: 2 },
      { provider: "openai", model: "gpt-5", score: 0, callCount: 4 },
    ]);

    expect(rows.map((row) => row.model)).toEqual(["gpt-5", "claude-sonnet-4-6", "claude-opus-4-6"]);
    expect(rows[0]?.lowestScore).toBe(0);
    expect(rows[1]?.cells.openrouter).toEqual({
      provider: "openrouter",
      model: "claude-sonnet-4-6",
      score: 50,
      callCount: 2,
      band: "degraded",
    });
    expect(rows[1]?.cells.openai.band).toBe("no-data");
    expect(rows[2]?.cells.anthropic).toEqual({
      provider: "anthropic",
      model: "claude-opus-4-6",
      score: undefined,
      callCount: 0,
      band: "no-data",
    });
    expect(modelStatusCallCountLabel(1)).toBe("1 call");
    expect(modelStatusCallCountLabel(3)).toBe("3 calls");
    expect(MODEL_STATUS_DEFAULT_WINDOW).toBe("5m");
  });
});
