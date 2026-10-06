import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { FirstRunArtPane } from "./FirstRunArtPane";
import type { FirstRunArtScene } from "./firstRunArtScene";

const welcome: FirstRunArtScene = {
  mode: "school",
  background: "#87ae9d",
  arrowColor: "#eeede9",
  pill: { label: "Discover", value: "Awaiting code", light: false },
};

const connect: FirstRunArtScene = {
  mode: "globe",
  background: "#b7b174",
  arrowColor: "#eeede9",
  count: 100,
  pill: { label: "Discover", value: "Awaiting code", light: false },
};

describe("FirstRunArtPane", () => {
  afterEach(() => {
    delete window.THREE;
    delete window.SuperplaneArt;
    document.querySelectorAll("script[data-onboarding-src]").forEach((script) => script.remove());
  });

  it("mounts the preview options, then tweens and mounts again on the next step", async () => {
    const destroy = vi.fn();
    const setColors = vi.fn((_background: string, _arrowColor: string, _durationMs: number, onDone?: () => void) => {
      onDone?.();
    });
    const mount = vi.fn(() => ({ setColors, destroy }));
    window.THREE = {};
    window.SuperplaneArt = { mount };

    const view = render(<FirstRunArtPane art={welcome} />);
    await waitFor(() => expect(mount).toHaveBeenCalled());
    expect(mount.mock.calls[0]?.[1]).toMatchObject({
      mode: "school",
      panel: false,
      minimalSphere: true,
      allowModeSwitch: false,
      bgColor: "#87ae9d",
      arrowColor: "#eeede9",
    });

    view.rerender(<FirstRunArtPane art={connect} />);

    await waitFor(() => expect(setColors).toHaveBeenCalledWith("#b7b174", "#eeede9", 1400, expect.any(Function)));
    await waitFor(() => expect(mount.mock.calls.length).toBeGreaterThan(1));
    expect(mount.mock.calls.at(-1)?.[1]).toMatchObject({
      mode: "globe",
      panel: false,
      minimalSphere: true,
      allowModeSwitch: false,
      overrides: { globe: { count: 100 } },
    });
    expect(screen.getByTestId("first-run-art-pane")).toHaveAttribute("data-art-count", "100");
  });
});
