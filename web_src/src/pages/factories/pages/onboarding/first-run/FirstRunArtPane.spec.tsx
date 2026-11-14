import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { FirstRunArtPane } from "./FirstRunArtPane";
import type { FirstRunArtScene } from "./firstRunArtScene";
import type { OnboardingArtMountOptions } from "./onboardingArt";
import * as onboardingArt from "./onboardingArt";
import { FirstRunShell } from "./FirstRunShell";
import { Button } from "@/components/ui/button";

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
    cleanup();
    vi.restoreAllMocks();
    delete window.THREE;
    delete window.SuperplaneArt;
    document.querySelectorAll("script[data-onboarding-src]").forEach((script) => script.remove());
  });

  it("mounts the preview options, then tweens and mounts again on the next step", async () => {
    const destroy = vi.fn();
    const replacementDestroy = vi.fn();
    let completeTween: (() => void) | undefined;
    const setColors = vi.fn((_background: string, _arrowColor: string, _durationMs: number, onDone?: () => void) => {
      completeTween = onDone;
    });
    const mount = vi.fn((_target: HTMLElement, _options: OnboardingArtMountOptions) => ({ setColors, destroy }));
    mount
      .mockImplementationOnce(() => ({ setColors, destroy }))
      .mockImplementation(() => ({ setColors, destroy: replacementDestroy }));
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
    expect(mount).toHaveBeenCalledTimes(1);
    act(() => completeTween?.());
    await waitFor(() => expect(mount.mock.calls.length).toBeGreaterThan(1));
    expect(mount.mock.calls.at(-1)?.[1]).toMatchObject({
      mode: "globe",
      panel: false,
      minimalSphere: true,
      allowModeSwitch: false,
      overrides: { globe: { count: 100 } },
    });
    expect(screen.getByTestId("first-run-art-pane")).toHaveAttribute("data-art-count", "100");
    expect(destroy).toHaveBeenCalledTimes(1);
    view.unmount();
    expect(replacementDestroy).toHaveBeenCalledTimes(1);
  });

  it("preserves the onboarding background, overlays, and controls when initial mounting throws", async () => {
    const mount = vi.fn((target: HTMLElement) => {
      target.append(document.createElement("canvas"));
      throw new Error("renderer initialization failed");
    });
    window.THREE = {};
    window.SuperplaneArt = { mount };
    render(
      <FirstRunShell testId="onboarding" visual="preview" sphere={{ level: 0, art: welcome }}>
        <Button type="button">Continue</Button>
      </FirstRunShell>,
    );

    await waitFor(() => expect(mount).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("first-run-art-canvas").childNodes).toHaveLength(0);
    expect(screen.getByTestId("first-run-art-stage")).toHaveStyle({ backgroundColor: welcome.background });
    expect(screen.getByText("Awaiting code")).toBeVisible();
    expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled();
  });

  it("retires the previous handle and preserves badges when a deferred transition mount throws", async () => {
    let completeTween: (() => void) | undefined;
    const destroy = vi.fn();
    const setColors = vi.fn((_background: string, _arrow: string, _duration: number, onDone?: () => void) => {
      completeTween = onDone;
    });
    const mount = vi.fn((_target: HTMLElement) => ({ setColors, destroy }));
    mount
      .mockImplementationOnce(() => ({ setColors, destroy }))
      .mockImplementation((target) => {
        target.append(document.createElement("canvas"));
        throw new Error("transition renderer failed");
      });
    window.THREE = {};
    window.SuperplaneArt = { mount };
    const view = render(<FirstRunArtPane art={welcome} />);
    await waitFor(() => expect(mount).toHaveBeenCalledTimes(1));
    view.rerender(
      <FirstRunArtPane
        art={{ ...connect, pill: undefined, badges: { discover: "Repository found", verify: "Ready" } }}
      />,
    );
    await waitFor(() => expect(setColors).toHaveBeenCalledTimes(1));

    expect(() => act(() => completeTween?.())).not.toThrow();
    expect(destroy).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("first-run-art-canvas").childNodes).toHaveLength(0);
    expect(screen.getByText("Repository found")).toBeVisible();
    expect(screen.getByText("Ready")).toBeVisible();
    view.unmount();
    expect(destroy).toHaveBeenCalledTimes(1);
    expect(mount).toHaveBeenCalledTimes(2);
  });

  it("does not mount after pending initialization is cancelled", async () => {
    let finishLoading!: () => void;
    vi.spyOn(onboardingArt, "loadOnboardingArt").mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          finishLoading = resolve;
        }),
    );
    const mount = vi.fn(() => ({ setColors: vi.fn(), destroy: vi.fn() }));
    window.SuperplaneArt = { mount };
    const view = render(<FirstRunArtPane art={welcome} />);
    view.unmount();

    await act(async () => finishLoading());
    expect(mount).not.toHaveBeenCalled();
  });
});
