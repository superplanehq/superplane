import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import { afterEach, describe, expect, it, vi } from "bun:test";

const assets = new URL("../../../../../../public/assets/onboarding/", import.meta.url);

describe("owned artwork context acquisition", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    delete window.THREE;
    delete window.SuperplaneArt;
  });

  for (const failure of ["null", "throw"] as const) {
    it(`leaves no opaque root or Three.js console error when context acquisition returns ${failure}`, () => {
      const sandbox: { window: { SuperplaneArt?: typeof window.SuperplaneArt }; document: Document; console: Console } =
        { window: {}, document, console };
      runInNewContext(readFileSync(new URL("three.min.js", assets), "utf8"), sandbox);
      runInNewContext(readFileSync(new URL("superplane-art.js", assets), "utf8"), sandbox);
      const getContext = vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockImplementation(() => {
        if (failure === "throw") throw new Error("context unavailable");
        return null;
      });
      const consoleError = vi.spyOn(console, "error");
      const target = document.createElement("div");

      expect(
        sandbox.window.SuperplaneArt?.mount(target, {
          mode: "school",
          panel: false,
          minimalSphere: true,
          allowModeSwitch: false,
          bgColor: "#87ae9d",
          arrowColor: "#eeede9",
        }),
      ).toBeNull();
      expect(getContext).toHaveBeenCalled();
      expect(target.childNodes).toHaveLength(0);
      expect(consoleError).not.toHaveBeenCalled();
    });
  }
});
