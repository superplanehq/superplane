import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useAgentNodeFocusRequest, type CanvasFocusRequest } from "./useAgentNodeFocusRequest";

describe("useAgentNodeFocusRequest", () => {
  it("turns agent:focus-node into a live canvas focus request", () => {
    const setFocusRequest = vi.fn<(request: CanvasFocusRequest) => void>();
    const targetModeRef = { current: "live" as const };
    renderHook(() => useAgentNodeFocusRequest(setFocusRequest, targetModeRef));

    act(() => {
      window.dispatchEvent(new CustomEvent("agent:focus-node", { detail: { nodeId: "http-fetch-abc123" } }));
    });

    expect(setFocusRequest).toHaveBeenCalledTimes(1);
    expect(setFocusRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        nodeId: "http-fetch-abc123",
        targetMode: "live",
        tab: "settings",
      }),
    );
    expect(setFocusRequest.mock.calls[0][0].requestId).toEqual(expect.any(Number));
  });

  it("targets the run canvas while run inspection is active", () => {
    const setFocusRequest = vi.fn<(request: CanvasFocusRequest) => void>();
    const targetModeRef = { current: "runs" as const };
    renderHook(() => useAgentNodeFocusRequest(setFocusRequest, targetModeRef));

    act(() => {
      window.dispatchEvent(new CustomEvent("agent:focus-node", { detail: { nodeId: "http-fetch-abc123" } }));
    });

    expect(setFocusRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        nodeId: "http-fetch-abc123",
        targetMode: "runs",
      }),
    );
    expect(setFocusRequest.mock.calls[0][0].tab).toBeUndefined();
  });

  it("ignores events without a node id", () => {
    const setFocusRequest = vi.fn<(request: CanvasFocusRequest) => void>();
    const targetModeRef = { current: "live" as const };
    renderHook(() => useAgentNodeFocusRequest(setFocusRequest, targetModeRef));

    act(() => {
      window.dispatchEvent(new CustomEvent("agent:focus-node", { detail: {} }));
    });

    expect(setFocusRequest).not.toHaveBeenCalled();
  });
});
