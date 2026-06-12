import { describe, expect, it } from "vitest";
import { getWorkflowViewPresentation } from "./viewState";

describe("getWorkflowViewPresentation", () => {
  it("keeps run inspection read-only until edit mode is active", () => {
    const inspectingRun = getWorkflowViewPresentation({
      isConsoleMode: false,
      isRunInspectionMode: true,
      isMemoryMode: false,
      isFilesMode: false,
      isVersionsMode: false,
      hasEditableVersion: false,
      isViewingPendingApprovalVersion: false,
      isViewingCurrentLiveVersion: true,
    });

    expect(inspectingRun.readOnlyViewModes).toBe(true);

    const editingFromRunLink = getWorkflowViewPresentation({
      isConsoleMode: false,
      isRunInspectionMode: true,
      isMemoryMode: false,
      isFilesMode: false,
      isVersionsMode: false,
      hasEditableVersion: true,
      isViewingPendingApprovalVersion: false,
      isViewingCurrentLiveVersion: true,
    });

    expect(editingFromRunLink.readOnlyViewModes).toBe(false);
  });
});
