import { describe, expect, it, vi } from "vitest";

import {
  requestCanvasAgentSidebarClose,
  requestCanvasAgentSidebarOpen,
  subscribeCanvasAgentSidebarOpen,
  subscribeCanvasAgentSidebarState,
} from "./canvasAgentSidebarOpenRequest";

describe("canvasAgentSidebarOpenRequest", () => {
  it("notifies subscribers for the requested canvas", () => {
    const onOpen = vi.fn();
    const unsubscribe = subscribeCanvasAgentSidebarOpen(onOpen);

    requestCanvasAgentSidebarOpen("canvas-a");
    unsubscribe();

    expect(onOpen).toHaveBeenCalledWith("canvas-a");
  });

  it("notifies state subscribers for close requests", () => {
    const onChange = vi.fn();
    const unsubscribe = subscribeCanvasAgentSidebarState(onChange);

    requestCanvasAgentSidebarClose("canvas-a");
    unsubscribe();

    expect(onChange).toHaveBeenCalledWith("canvas-a", false);
  });

  it("ignores empty canvas ids", () => {
    const onOpen = vi.fn();
    const unsubscribe = subscribeCanvasAgentSidebarOpen(onOpen);

    requestCanvasAgentSidebarOpen("");
    unsubscribe();

    expect(onOpen).not.toHaveBeenCalled();
  });
});
