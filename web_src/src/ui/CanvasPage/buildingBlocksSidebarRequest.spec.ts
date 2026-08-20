import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  clearBuildingBlocksSidebarRequests,
  publishBuildingBlocksSidebarChanged,
  requestBuildingBlocksSidebar,
  subscribeBuildingBlocksSidebarChanged,
  subscribeBuildingBlocksSidebarRequest,
  useBuildingBlocksSidebarRequest,
} from "./buildingBlocksSidebarRequest";

describe("buildingBlocksSidebarRequest", () => {
  beforeEach(() => {
    clearBuildingBlocksSidebarRequests();
  });

  it("notifies request subscribers", () => {
    const onChange = vi.fn();
    const unsubscribe = subscribeBuildingBlocksSidebarRequest(onChange);

    requestBuildingBlocksSidebar("canvas-a", true);
    unsubscribe();

    expect(onChange).toHaveBeenCalledWith("canvas-a", true);
  });

  it("notifies change subscribers without notifying request subscribers", () => {
    const onRequest = vi.fn();
    const onChanged = vi.fn();
    const unsubscribeRequest = subscribeBuildingBlocksSidebarRequest(onRequest);
    const unsubscribeChanged = subscribeBuildingBlocksSidebarChanged(onChanged);

    publishBuildingBlocksSidebarChanged("canvas-a", false);
    unsubscribeRequest();
    unsubscribeChanged();

    expect(onRequest).not.toHaveBeenCalled();
    expect(onChanged).toHaveBeenCalledWith("canvas-a", false);
  });

  it("replays the last request when the hook mounts after the event", () => {
    requestBuildingBlocksSidebar("canvas-late", true);
    const onToggle = vi.fn();

    renderHook(() => useBuildingBlocksSidebarRequest("canvas-late", onToggle));

    expect(onToggle).toHaveBeenCalledWith(true);
  });

  it("does not replay after requests are cleared", () => {
    requestBuildingBlocksSidebar("canvas-late", true);
    clearBuildingBlocksSidebarRequests();
    const onToggle = vi.fn();

    renderHook(() => useBuildingBlocksSidebarRequest("canvas-late", onToggle));

    expect(onToggle).not.toHaveBeenCalled();
  });
});
