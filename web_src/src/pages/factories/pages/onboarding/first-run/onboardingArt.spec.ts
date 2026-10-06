import { describe, expect, it } from "bun:test";

import type { FirstRunArtScene } from "./firstRunArtScene";
import { onboardingArtMountOptions } from "./onboardingArt";

const globe: FirstRunArtScene = {
  mode: "globe",
  background: "#b7b174",
  arrowColor: "#eeede9",
  count: 100,
};

describe("onboardingArtMountOptions", () => {
  it("mounts the owned script with the preview options", () => {
    expect(onboardingArtMountOptions({ mode: "school", background: "#87ae9d", arrowColor: "#eeede9" })).toEqual({
      mode: "school",
      panel: false,
      minimalSphere: true,
      allowModeSwitch: false,
      bgColor: "#87ae9d",
      arrowColor: "#eeede9",
    });
    expect(onboardingArtMountOptions(globe)).toMatchObject({
      panel: false,
      minimalSphere: true,
      allowModeSwitch: false,
      overrides: { globe: { count: 100 } },
    });
  });
});
