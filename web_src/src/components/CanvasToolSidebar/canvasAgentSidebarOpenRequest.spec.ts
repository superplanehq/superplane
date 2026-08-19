import { describe, expect, it, vi } from "vitest";

import { requestCanvasAgentSidebarOpen, subscribeCanvasAgentSidebarOpen } from "./canvasAgentSidebarOpenRequest";

describe("canvasAgentSidebarOpenRequest", () => {
  it("notifies subscribers for the requested canvas", () => {
    const onOpen = vi.fn();
    const unsubscribe = subscribeCanvasAgentSidebarOpen(onOpen);

    requestCanvasAgentSidebarOpen("canvas-a");
    unsubscribe();

    expect(onOpen).toHaveBeenCalledWith("canvas-a");
  });

  it("ignores empty canvas ids", () => {
    const onOpen = vi.fn();
    const unsubscribe = subscribeCanvasAgentSidebarOpen(onOpen);

    requestCanvasAgentSidebarOpen("");
    unsubscribe();

    expect(onOpen).not.toHaveBeenCalled();
  });
});
