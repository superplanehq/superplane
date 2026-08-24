import { describe, expect, it } from "vitest";

import {
  setParallelismLabel,
  clampLineStepParallelism,
  DEFAULT_LINE_STEP_PARALLELISM,
  lineStepParallelism,
  replaceLineStepParallelism,
} from "./factoryLineFormShared";

describe("lineStepParallelism", () => {
  it("uses 10 when the step has no value", () => {
    expect(lineStepParallelism(undefined)).toBe(DEFAULT_LINE_STEP_PARALLELISM);
    expect(lineStepParallelism({})).toBe(10);
    expect(lineStepParallelism({ maxParallelism: 25 })).toBe(25);
  });

  it("clamps values to 1–100", () => {
    expect(clampLineStepParallelism(0)).toBe(1);
    expect(clampLineStepParallelism(140)).toBe(100);
    expect(clampLineStepParallelism(12.6)).toBe(13);
  });

  it("names the menu item with the current value", () => {
    expect(setParallelismLabel(10)).toBe("Set parallelism (10)");
  });

  it("writes parallelism onto one step and keeps the others", () => {
    const steps = replaceLineStepParallelism(
      [
        { type: "runApp", app: { app: "app-plan", entrypoint: "start-plan" } },
        { type: "runApp", app: { app: "app-impl", entrypoint: "start-impl" }, maxParallelism: 4 },
      ],
      0,
      20,
    );

    expect(steps[0]?.maxParallelism).toBe(20);
    expect(steps[1]?.maxParallelism).toBe(4);
  });
});
