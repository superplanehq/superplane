import { describe, expect, it } from "vitest";
import {
  allowsRunsSidebar,
  applyRunInspectionNavigationSearchParams,
  clampWorkflowViewFlagsForFactoryApp,
  clearRunInspectionSearchParams,
  getExitEditModeDisabledTooltip,
  resolveCanvasPageInitialSidebar,
  getWorkflowViewPresentation,
  isNonCanvasAppViewParam,
} from "./viewState";

describe("clampWorkflowViewFlagsForFactoryApp", () => {
  it("forces canvas-only flags for factory apps", () => {
    expect(
      clampWorkflowViewFlagsForFactoryApp({
        isRunInspectionMode: true,
        isMemoryMode: true,
        isFilesMode: true,
        isConsoleMode: true,
      }),
    ).toEqual({
      isRunInspectionMode: true,
      isMemoryMode: false,
      isFilesMode: false,
      isConsoleMode: false,
    });
  });
});

describe("isNonCanvasAppViewParam", () => {
  it("detects console, memory, and files views", () => {
    expect(isNonCanvasAppViewParam("console")).toBe(true);
    expect(isNonCanvasAppViewParam("dashboard")).toBe(true);
    expect(isNonCanvasAppViewParam("memory")).toBe(true);
    expect(isNonCanvasAppViewParam("files")).toBe(true);
    expect(isNonCanvasAppViewParam("")).toBe(false);
    expect(isNonCanvasAppViewParam("runs")).toBe(false);
  });
});

describe("allowsRunsSidebar", () => {
  it("allows the runs sidebar on the Canvas workflow tab", () => {
    expect(allowsRunsSidebar(undefined)).toBe(true);
    expect(allowsRunsSidebar("default")).toBe(true);
    expect(allowsRunsSidebar("version-live")).toBe(true);
  });

  it("allows the runs sidebar on the Console tab", () => {
    expect(allowsRunsSidebar("console")).toBe(true);
  });

  it("hides the runs sidebar on the Memory and Files surfaces", () => {
    expect(allowsRunsSidebar("memory")).toBe(false);
    expect(allowsRunsSidebar("files")).toBe(false);
  });
});

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

  it("keeps the component editor selection when Configure is entering", () => {
    const next = clearRunInspectionSearchParams(
      new URLSearchParams({
        run: "run-42",
        configure: "1",
        sidebar: "1",
        node: "create-pr",
      }),
    );

    expect(next.get("run")).toBeNull();
    expect(next.get("configure")).toBe("1");
    expect(next.get("sidebar")).toBe("1");
    expect(next.get("node")).toBe("create-pr");
  });
});

describe("resolveCanvasPageInitialSidebar", () => {
  it("opens the component editor from the URL during Configure even if a run is still present", () => {
    expect(
      resolveCanvasPageInitialSidebar({
        factoryConfigure: true,
        runInspectionChromeActive: true,
        searchParams: new URLSearchParams("run=run-42&configure=1&sidebar=1&node=create-pr"),
      }),
    ).toEqual({ isOpen: true, nodeId: "create-pr" });
  });

  it("does not restore the live node inspector from run inspection params", () => {
    expect(
      resolveCanvasPageInitialSidebar({
        factoryConfigure: false,
        runInspectionChromeActive: true,
        searchParams: new URLSearchParams("run=run-42&sidebar=1&node=create-pr"),
      }),
    ).toEqual({ isOpen: false, nodeId: null });
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

describe("getExitEditModeDisabledTooltip", () => {
  it("prioritizes remote deletion over permission denial", () => {
    expect(
      getExitEditModeDisabledTooltip({
        canUpdateCanvas: false,
        canvasDeletedRemotely: true,
        hasEditableVersion: true,
      }),
    ).toBe("This canvas was deleted in another session.");
  });
});
