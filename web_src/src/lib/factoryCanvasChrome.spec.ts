import { describe, expect, it } from "vitest";
import {
  factoryCanvasBackground,
  factoryEdgePalette,
  factoryEdgeToneClassName,
  primaryEventStateFromCanvasNodeData,
  resolveFactoryEdgeTone,
} from "./factoryCanvasChrome";

describe("factoryCanvasChrome", () => {
  it("matches Storybook dark canvas dots and background", () => {
    expect(factoryCanvasBackground(true)).toEqual({
      gap: 22,
      size: 1,
      color: "#33312b",
      bgColor: "#14120b",
    });
  });

  it("uses 3px dots while editing so the grid reads as the edit surface", () => {
    expect(factoryCanvasBackground(true, true)).toEqual({
      gap: 22,
      size: 3,
      color: "#33312b",
      bgColor: "#14120b",
    });
    expect(factoryCanvasBackground(false, true)).toMatchObject({ gap: 22, size: 3, color: "#e5e7eb" });
  });

  it("uses thin slate edges in light mode", () => {
    expect(factoryEdgePalette(false).default).toEqual({ stroke: "#cbd5e1", strokeWidth: 1.5 });
  });

  it("maps event state to edge tone and class", () => {
    expect(resolveFactoryEdgeTone("running")).toBe("running");
    expect(resolveFactoryEdgeTone("cancelling")).toBe("running");
    expect(resolveFactoryEdgeTone("failed")).toBe("failed");
    expect(resolveFactoryEdgeTone("error")).toBe("failed");
    expect(resolveFactoryEdgeTone("success")).toBe("default");
    expect(factoryEdgeToneClassName("running")).toBe("sp-edge-factory-running");
    expect(factoryEdgeToneClassName("failed")).toBe("sp-edge-factory-failed");
    expect(factoryEdgeToneClassName("default")).toBeUndefined();
  });

  it("reads primary event state from component or trigger", () => {
    expect(
      primaryEventStateFromCanvasNodeData({
        component: { eventSections: [{ eventState: "running" }] },
      }),
    ).toBe("running");
    expect(
      primaryEventStateFromCanvasNodeData({
        trigger: { eventSections: [{ eventState: "failed" }] },
      }),
    ).toBe("failed");
    expect(primaryEventStateFromCanvasNodeData({})).toBeUndefined();
  });
});
