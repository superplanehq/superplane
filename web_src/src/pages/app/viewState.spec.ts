import { describe, expect, it } from "vitest";
import {
  applyRunInspectionNavigationSearchParams,
  clearRunInspectionSearchParams,
  getWorkflowViewPresentation,
} from "./viewState";

describe("clearRunInspectionSearchParams", () => {
  it("removes run inspection params from the URL", () => {
    const next = clearRunInspectionSearchParams(
      new URLSearchParams({ run: "run-42", sidebar: "1", node: "node-1", version: "draft-1" }),
    );

    expect(next.get("run")).toBeNull();
    expect(next.get("sidebar")).toBeNull();
    expect(next.get("node")).toBeNull();
    expect(next.get("version")).toBe("draft-1");
  });
});

describe("applyRunInspectionNavigationSearchParams", () => {
  it("clears console view when opening run inspection", () => {
    const next = applyRunInspectionNavigationSearchParams(
      new URLSearchParams({ view: "console", sidebar: "1", node: "old-node" }),
      { runId: "run-42", nodeId: "failed-node" },
    );

    expect(next.get("view")).toBeNull();
    expect(next.get("run")).toBe("run-42");
    expect(next.get("sidebar")).toBe("1");
    expect(next.get("node")).toBe("failed-node");
  });
});

describe("getWorkflowViewPresentation", () => {
  it("keeps run inspection read-only even when a draft is active", () => {
    const inspectingRun = getWorkflowViewPresentation({
      isConsoleMode: false,
      isRunInspectionMode: true,
      isMemoryMode: false,
      isFilesMode: false,
      hasEditableVersion: false,
      isViewingCurrentLiveVersion: true,
    });

    expect(inspectingRun.readOnlyViewModes).toBe(true);

    const editingAfterRunInspection = getWorkflowViewPresentation({
      isConsoleMode: false,
      isRunInspectionMode: false,
      isMemoryMode: false,
      isFilesMode: false,
      hasEditableVersion: true,
      isViewingCurrentLiveVersion: true,
    });

    expect(editingAfterRunInspection.readOnlyViewModes).toBe(false);
    expect(editingAfterRunInspection.hideAddControls).toBe(false);
  });
});
