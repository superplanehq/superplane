import { readFileSync } from "node:fs";
import { createContext, runInContext } from "node:vm";
import { afterEach, describe, expect, it, vi } from "bun:test";

import type { OnboardingArtHandle, OnboardingArtMountOptions } from "./onboardingArt";

const assets = new URL("../../../../../../public/assets/onboarding/", import.meta.url);
const threeScript = readFileSync(new URL("three.min.js", assets), "utf8");
const artScript = readFileSync(new URL("superplane-art.js", assets), "utf8");

describe("onboarding artwork context acquisition", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    document.getElementById("sp-art-thumb-css")?.remove();
  });

  for (const unavailable of ["null", "exception"] as const) {
    it(`returns null without renderer initialization or covering artwork when context acquisition returns ${unavailable}`, () => {
      const getContext = vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockImplementation(() => {
        if (unavailable === "exception") throw new Error("graphics disabled by browser");
        return null;
      });
      const error = vi.fn();
      const retry = vi.fn();
      const sandbox = createContext({ document, console: { ...console, error }, setInterval: retry, window: {} });
      runInContext(threeScript, sandbox);
      const three = sandbox.THREE as { WebGLRenderer: (...args: unknown[]) => unknown };
      const renderer = vi.spyOn(three, "WebGLRenderer");
      runInContext(artScript, sandbox);
      const art = sandbox.window.SuperplaneArt as {
        mount: (target: HTMLElement, options: Partial<OnboardingArtMountOptions>) => OnboardingArtHandle | null;
      };
      const container = document.createElement("div");
      const fallback = document.createElement("img");
      container.appendChild(fallback);

      expect(art.mount(container, { mode: "school", panel: false })).toBeNull();
      expect(getContext).toHaveBeenCalled();
      expect(renderer).not.toHaveBeenCalled();
      expect(Array.from(container.children)).toEqual([fallback]);
      expect(error).not.toHaveBeenCalled();
      expect(retry).not.toHaveBeenCalled();
    });
  }
});
