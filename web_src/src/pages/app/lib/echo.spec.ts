import { describe, expect, it, vi } from "vitest";

import { armCreateDraftEcho, consumeCreateDraftEcho, registerCreateDraftEcho, type CreateDraftEchoMap } from "./echo";

describe("create draft echo guards", () => {
  it("consumes in-flight create draft echoes before the version id is armed", () => {
    vi.useFakeTimers();

    const echoMap = { current: new Map() as CreateDraftEchoMap };
    const canvasSaveSessionRef = { current: 1 };
    registerCreateDraftEcho(echoMap, canvasSaveSessionRef, "canvas-1");

    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "draft-version-1")).toBe(true);
    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "draft-version-1")).toBe(false);

    vi.useRealTimers();
  });

  it("consumes armed create draft echoes by version id", () => {
    vi.useFakeTimers();

    const echoMap = { current: new Map() as CreateDraftEchoMap };
    const canvasSaveSessionRef = { current: 1 };
    const release = registerCreateDraftEcho(echoMap, canvasSaveSessionRef, "canvas-1");

    armCreateDraftEcho(echoMap, "canvas-1", "draft-version-1", release);
    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "draft-version-1")).toBe(true);
    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "draft-version-1")).toBe(false);

    vi.useRealTimers();
  });

  it("ignores websocket events for other versions on the same canvas", () => {
    vi.useFakeTimers();

    const echoMap = { current: new Map() as CreateDraftEchoMap };
    const canvasSaveSessionRef = { current: 1 };
    const release = registerCreateDraftEcho(echoMap, canvasSaveSessionRef, "canvas-1");

    armCreateDraftEcho(echoMap, "canvas-1", "draft-version-1", release);

    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "other-version")).toBe(false);
    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "draft-version-1")).toBe(true);

    vi.useRealTimers();
  });

  it("arms the release returned from the matching onMutate registration", () => {
    vi.useFakeTimers();

    const echoMap = { current: new Map() as CreateDraftEchoMap };
    const canvasSaveSessionRef = { current: 1 };
    const firstRelease = registerCreateDraftEcho(echoMap, canvasSaveSessionRef, "canvas-1");
    const secondRelease = registerCreateDraftEcho(echoMap, canvasSaveSessionRef, "canvas-1");

    armCreateDraftEcho(echoMap, "canvas-1", "draft-version-2", secondRelease);

    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "draft-version-1")).toBe(false);
    expect(consumeCreateDraftEcho(echoMap, "canvas-1", "draft-version-2")).toBe(true);
    expect(typeof firstRelease).toBe("function");

    vi.useRealTimers();
  });
});
