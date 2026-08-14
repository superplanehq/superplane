import { describe, expect, it } from "vitest";
import { resolveShowNodeRuntimeActivity } from "./resolveShowNodeRuntimeActivity";

describe("resolveShowNodeRuntimeActivity", () => {
  it("follows showLiveActivity for non-factory canvases", () => {
    expect(
      resolveShowNodeRuntimeActivity({
        showLiveActivity: true,
        factoryEmbed: false,
        isRunInspectionMode: false,
      }),
    ).toBe(true);
    expect(
      resolveShowNodeRuntimeActivity({
        showLiveActivity: false,
        factoryEmbed: false,
        isRunInspectionMode: false,
      }),
    ).toBe(false);
  });

  it("hides factory Live activity and keeps run-scoped activity", () => {
    expect(
      resolveShowNodeRuntimeActivity({
        showLiveActivity: true,
        factoryEmbed: true,
        isRunInspectionMode: false,
      }),
    ).toBe(false);
    expect(
      resolveShowNodeRuntimeActivity({
        showLiveActivity: true,
        factoryEmbed: true,
        isRunInspectionMode: true,
      }),
    ).toBe(true);
  });
});
