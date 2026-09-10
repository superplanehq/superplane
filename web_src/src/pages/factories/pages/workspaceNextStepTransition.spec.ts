import { afterEach, describe, expect, it, vi } from "vitest";

import { runWorkspaceNextStepTransition } from "./workspaceNextStepTransition";

describe("runWorkspaceNextStepTransition", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("updates immediately when view transitions are not available", () => {
    const update = vi.fn();
    vi.stubGlobal("matchMedia", undefined);
    Object.defineProperty(document, "startViewTransition", { value: undefined, configurable: true });

    runWorkspaceNextStepTransition(update);

    expect(update).toHaveBeenCalledOnce();
  });

  it("skips the animation when the user prefers reduced motion", () => {
    const update = vi.fn();
    const startViewTransition = vi.fn();
    vi.stubGlobal("matchMedia", () => ({ matches: true }));
    Object.defineProperty(document, "startViewTransition", { value: startViewTransition, configurable: true });

    runWorkspaceNextStepTransition(update);

    expect(update).toHaveBeenCalledOnce();
    expect(startViewTransition).not.toHaveBeenCalled();
  });

  it("runs the update inside a view transition", () => {
    const update = vi.fn();
    const startViewTransition = vi.fn((callback: () => void) => {
      callback();
      return { finished: Promise.resolve() };
    });
    vi.stubGlobal("matchMedia", () => ({ matches: false }));
    Object.defineProperty(document, "startViewTransition", { value: startViewTransition, configurable: true });

    runWorkspaceNextStepTransition(update);

    expect(startViewTransition).toHaveBeenCalledOnce();
    expect(update).toHaveBeenCalledOnce();
  });
});
