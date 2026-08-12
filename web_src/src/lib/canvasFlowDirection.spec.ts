import { describe, expect, it } from "vitest";
import { isFactoryApp, isVerticalCanvasFlow, resolveCanvasFlowDirection } from "./canvasFlowDirection";

describe("isFactoryApp", () => {
  it("detects factory-owned apps", () => {
    expect(isFactoryApp("factory-1")).toBe(true);
    expect(isFactoryApp(undefined)).toBe(false);
    expect(isFactoryApp("")).toBe(false);
  });
});

describe("resolveCanvasFlowDirection", () => {
  it("uses vertical flow for factory apps", () => {
    expect(resolveCanvasFlowDirection("factory-1")).toBe("vertical");
  });

  it("uses horizontal flow for regular apps", () => {
    expect(resolveCanvasFlowDirection(undefined)).toBe("horizontal");
    expect(resolveCanvasFlowDirection(null)).toBe("horizontal");
    expect(resolveCanvasFlowDirection("")).toBe("horizontal");
  });
});

describe("isVerticalCanvasFlow", () => {
  it("detects vertical direction", () => {
    expect(isVerticalCanvasFlow("vertical")).toBe(true);
    expect(isVerticalCanvasFlow("horizontal")).toBe(false);
    expect(isVerticalCanvasFlow(undefined)).toBe(false);
  });
});
