import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  AGENT_BOOT_CONTEXT_READY_EVENT,
  abandonPendingPlaceholderBoot,
  getAgentBootMessage,
  isAgentBootReady,
  markAgentBootReady,
  PLACEHOLDER_NODE_CONTEXT_KEY,
  setAgentBootContext,
} from "./agentBootContext";

describe("agent boot context", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("blocks boot while a placeholder node is pending for the canvas", () => {
    sessionStorage.setItem(PLACEHOLDER_NODE_CONTEXT_KEY, "canvas-1");

    expect(isAgentBootReady("canvas-1")).toBe(false);
    expect(isAgentBootReady("canvas-2")).toBe(true);
  });

  it("marks boot ready after the placeholder node is created", () => {
    const listener = vi.fn();
    window.addEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, listener);
    sessionStorage.setItem(PLACEHOLDER_NODE_CONTEXT_KEY, "canvas-1");

    markAgentBootReady("canvas-1");

    expect(isAgentBootReady("canvas-1")).toBe(true);
    expect(sessionStorage.getItem(PLACEHOLDER_NODE_CONTEXT_KEY)).toBeNull();
    expect(listener).toHaveBeenCalledWith(expect.objectContaining({ detail: { canvasId: "canvas-1" } }));

    window.removeEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, listener);
  });

  it("abandons pending placeholder boot when the placeholder cannot be created", () => {
    const listener = vi.fn();
    window.addEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, listener);
    sessionStorage.setItem(PLACEHOLDER_NODE_CONTEXT_KEY, "canvas-1");
    setAgentBootContext("canvas-1", "blank");

    abandonPendingPlaceholderBoot("canvas-1");

    expect(isAgentBootReady("canvas-1")).toBe(true);
    expect(sessionStorage.getItem(PLACEHOLDER_NODE_CONTEXT_KEY)).toBeNull();
    expect(getAgentBootMessage("canvas-1")).toBe(
      "Session ready. Read the current canvas state, check connected integrations, and greet the user.",
    );
    expect(listener).toHaveBeenCalledWith(expect.objectContaining({ detail: { canvasId: "canvas-1" } }));

    window.removeEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, listener);
  });
});
