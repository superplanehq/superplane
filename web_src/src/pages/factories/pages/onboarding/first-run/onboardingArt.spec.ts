import { describe, expect, it } from "bun:test";

import type { FirstRunArtScene } from "./firstRunArtScene";
import { onboardingArtMountOptions, onboardingArtScriptUrl } from "./onboardingArt";

const globe: FirstRunArtScene = {
  mode: "globe",
  background: "#b7b174",
  arrowColor: "#eeede9",
  count: 100,
};

describe("onboardingArtScriptUrl", () => {
  it("requests the scripts from the asset route the production server serves", () => {
    expect(onboardingArtScriptUrl("three.min.js")).toBe("/assets/onboarding/three.min.js");
    expect(onboardingArtScriptUrl("superplane-art.js", "/app")).toBe("/app/assets/onboarding/superplane-art.js");
  });
});

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
