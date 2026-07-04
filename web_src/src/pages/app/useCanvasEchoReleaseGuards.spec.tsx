import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { CanvasEchoRelease } from "./canvasSaveTypes";
import { useCanvasEchoReleaseGuards } from "./useCanvasEchoReleaseGuards";

function renderEchoGuards() {
  const canvasSaveSessionRef = { current: 1 };
  const ignoredCanvasUpdatedEchoReleasesRef = { current: [] as CanvasEchoRelease[] };
  const ignoredCanvasVersionUpdatedEchoReleasesRef = { current: new Map<string, CanvasEchoRelease[]>() };

  const hook = renderHook(() =>
    useCanvasEchoReleaseGuards({
      canvasSaveSessionRef,
      ignoredCanvasUpdatedEchoReleasesRef,
      ignoredCanvasVersionUpdatedEchoReleasesRef,
    }),
  );

  return {
    ...hook,
    canvasSaveSessionRef,
    ignoredCanvasUpdatedEchoReleasesRef,
    ignoredCanvasVersionUpdatedEchoReleasesRef,
  };
}

describe("useCanvasEchoReleaseGuards", () => {
  it("consumes a registered canvas_version_updated echo for the matching version", () => {
    vi.useFakeTimers();

    const { result } = renderEchoGuards();

    act(() => {
      result.current.registerIgnoredCanvasVersionUpdatedEcho("draft-version-1");
    });

    expect(result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1")).toBe(true);
    expect(result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1")).toBe(false);

    vi.useRealTimers();
  });

  it("consumes a create-draft echo only for the armed version", () => {
    vi.useFakeTimers();

    const { result } = renderEchoGuards();

    let release: (() => void) | undefined;
    act(() => {
      release = result.current.registerIgnoredCreateDraftEcho("canvas-1");
      result.current.armIgnoredCreateDraftEcho("canvas-1", "draft-version-1", release!);
    });

    expect(result.current.consumeIgnoredCreateDraftEcho("canvas-1", "other-version")).toBe(false);
    expect(result.current.consumeIgnoredCreateDraftEcho("canvas-1", "draft-version-1")).toBe(true);
    expect(result.current.consumeIgnoredCreateDraftEcho("canvas-1", "draft-version-1")).toBe(false);

    vi.useRealTimers();
  });

  it("clears both create-draft and version echoes when both were registered for one create", () => {
    vi.useFakeTimers();

    const { result } = renderEchoGuards();

    let release: (() => void) | undefined;
    act(() => {
      release = result.current.registerIgnoredCreateDraftEcho("canvas-1");
      result.current.armIgnoredCreateDraftEcho("canvas-1", "draft-version-1", release!);
      result.current.registerIgnoredCanvasVersionUpdatedEcho("draft-version-1");
    });

    let consumedCreateDraftEcho = false;
    let consumedVersionEcho = false;
    act(() => {
      consumedCreateDraftEcho = result.current.consumeIgnoredCreateDraftEcho("canvas-1", "draft-version-1");
      consumedVersionEcho = result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1");
    });

    expect(consumedCreateDraftEcho).toBe(true);
    expect(consumedVersionEcho).toBe(true);
    expect(result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1")).toBe(false);

    vi.useRealTimers();
  });
});
