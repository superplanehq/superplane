import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { CanvasEchoRelease } from "./canvasSaveTypes";
import { useCanvasEchoReleaseGuards } from "./useCanvasEchoReleaseGuards";

function renderEchoGuards() {
  const canvasSaveSessionRef = { current: 1 };
  const ignoredCanvasUpdatedEchoReleasesRef = { current: [] as CanvasEchoRelease[] };

  const hook = renderHook(() =>
    useCanvasEchoReleaseGuards({
      canvasSaveSessionRef,
      ignoredCanvasUpdatedEchoReleasesRef,
    }),
  );

  return {
    ...hook,
    canvasSaveSessionRef,
    ignoredCanvasUpdatedEchoReleasesRef,
  };
}

describe("useCanvasEchoReleaseGuards", () => {
  it("consumes a registered canvas_updated echo once", () => {
    vi.useFakeTimers();

    const { result } = renderEchoGuards();

    act(() => {
      result.current.registerIgnoredCanvasUpdatedEcho();
    });

    expect(result.current.consumeIgnoredCanvasUpdatedEcho()).toBe(true);
    expect(result.current.consumeIgnoredCanvasUpdatedEcho()).toBe(false);

    vi.useRealTimers();
  });
});
