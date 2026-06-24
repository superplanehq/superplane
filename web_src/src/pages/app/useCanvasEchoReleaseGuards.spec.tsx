import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { CanvasEchoRelease } from "./canvasSaveTypes";
import { useCanvasEchoReleaseGuards } from "./useCanvasEchoReleaseGuards";

describe("useCanvasEchoReleaseGuards", () => {
  it("consumes a registered canvas_version_updated echo for the matching version", () => {
    vi.useFakeTimers();

    const canvasSaveSessionRef = { current: 1 };
    const ignoredCanvasUpdatedEchoReleasesRef = { current: [] as CanvasEchoRelease[] };
    const ignoredCanvasVersionUpdatedEchoReleasesRef = { current: new Map<string, CanvasEchoRelease[]>() };
    const ignoredCreateDraftEchoReleasesRef = { current: new Map<string, CanvasEchoRelease[]>() };

    const { result } = renderHook(() =>
      useCanvasEchoReleaseGuards({
        canvasSaveSessionRef,
        ignoredCanvasUpdatedEchoReleasesRef,
        ignoredCanvasVersionUpdatedEchoReleasesRef,
        ignoredCreateDraftEchoReleasesRef,
      }),
    );

    act(() => {
      result.current.registerIgnoredCanvasVersionUpdatedEcho("draft-version-1");
    });

    expect(result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1")).toBe(true);
    expect(result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1")).toBe(false);

    vi.useRealTimers();
  });

  it("consumes a registered create-draft echo for the matching canvas", () => {
    vi.useFakeTimers();

    const canvasSaveSessionRef = { current: 1 };
    const ignoredCanvasUpdatedEchoReleasesRef = { current: [] as CanvasEchoRelease[] };
    const ignoredCanvasVersionUpdatedEchoReleasesRef = { current: new Map<string, CanvasEchoRelease[]>() };
    const ignoredCreateDraftEchoReleasesRef = { current: new Map<string, CanvasEchoRelease[]>() };

    const { result } = renderHook(() =>
      useCanvasEchoReleaseGuards({
        canvasSaveSessionRef,
        ignoredCanvasUpdatedEchoReleasesRef,
        ignoredCanvasVersionUpdatedEchoReleasesRef,
        ignoredCreateDraftEchoReleasesRef,
      }),
    );

    act(() => {
      result.current.registerIgnoredCreateDraftEcho("canvas-1");
    });

    expect(result.current.consumeIgnoredCreateDraftEcho("canvas-1")).toBe(true);
    expect(result.current.consumeIgnoredCreateDraftEcho("canvas-1")).toBe(false);

    vi.useRealTimers();
  });

  it("clears both create-draft and version echoes when both were registered for one create", () => {
    vi.useFakeTimers();

    const canvasSaveSessionRef = { current: 1 };
    const ignoredCanvasUpdatedEchoReleasesRef = { current: [] as CanvasEchoRelease[] };
    const ignoredCanvasVersionUpdatedEchoReleasesRef = { current: new Map<string, CanvasEchoRelease[]>() };
    const ignoredCreateDraftEchoReleasesRef = { current: new Map<string, CanvasEchoRelease[]>() };

    const { result } = renderHook(() =>
      useCanvasEchoReleaseGuards({
        canvasSaveSessionRef,
        ignoredCanvasUpdatedEchoReleasesRef,
        ignoredCanvasVersionUpdatedEchoReleasesRef,
        ignoredCreateDraftEchoReleasesRef,
      }),
    );

    act(() => {
      result.current.registerIgnoredCreateDraftEcho("canvas-1");
      result.current.registerIgnoredCanvasVersionUpdatedEcho("draft-version-1");
    });

    let consumedCreateDraftEcho = false;
    let consumedVersionEcho = false;
    act(() => {
      consumedCreateDraftEcho = result.current.consumeIgnoredCreateDraftEcho("canvas-1");
      consumedVersionEcho = result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1");
    });

    expect(consumedCreateDraftEcho).toBe(true);
    expect(consumedVersionEcho).toBe(true);
    expect(result.current.consumeIgnoredCanvasVersionUpdatedEcho("draft-version-1")).toBe(false);

    vi.useRealTimers();
  });
});
