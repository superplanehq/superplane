import { describe, expect, it } from "vitest";
import { applyRunInspectionNavigationSearchParams, getWorkflowViewPresentation } from "./viewState";

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

    const inspectingRunWhileEditingDraft = getWorkflowViewPresentation({
      isConsoleMode: false,
      isRunInspectionMode: true,
      isMemoryMode: false,
      isFilesMode: false,
      hasEditableVersion: true,
      isViewingCurrentLiveVersion: true,
    });

    expect(inspectingRunWhileEditingDraft.readOnlyViewModes).toBe(true);
  });
});
