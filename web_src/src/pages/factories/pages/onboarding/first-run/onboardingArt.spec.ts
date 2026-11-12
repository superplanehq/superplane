import { afterEach, describe, expect, it, vi } from "bun:test";

import type { FirstRunArtScene } from "./firstRunArtScene";
import { mountOnboardingArt, onboardingArtMountOptions, onboardingArtScriptUrl } from "./onboardingArt";

const globe: FirstRunArtScene = {
  mode: "globe",
  background: "#b7b174",
  arrowColor: "#eeede9",
  count: 100,
};

afterEach(() => {
  delete window.SuperplaneArt;
});

describe("mountOnboardingArt", () => {
  it("removes only incomplete artwork when mounting throws", () => {
    const target = document.createElement("div");
    const existing = document.createElement("span");
    target.append(existing);
    const sibling = document.createElement("div");
    document.body.append(target, sibling);
    window.SuperplaneArt = {
      mount: (container) => {
        container.append(document.createElement("canvas"));
        throw new Error("renderer initialization failed");
      },
    };

    expect(mountOnboardingArt(target, globe)).toBeNull();
    expect(target.childNodes).toHaveLength(1);
    expect(existing.parentNode).toBe(target);
    expect(sibling.isConnected).toBe(true);
    target.remove();
    sibling.remove();
  });

  it("retains successfully mounted artwork and its lifecycle handle", () => {
    const target = document.createElement("div");
    const canvas = document.createElement("canvas");
    const handle = { setColors: vi.fn(), destroy: vi.fn() };
    window.SuperplaneArt = {
      mount: (container) => {
        container.append(canvas);
        return handle;
      },
    };

    expect(mountOnboardingArt(target, globe)).toBe(handle);
    expect(canvas.parentNode).toBe(target);
  });
});

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
