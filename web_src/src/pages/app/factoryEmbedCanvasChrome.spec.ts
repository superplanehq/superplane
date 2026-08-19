import { describe, expect, it, vi } from "vitest";
import type { RefObject } from "react";
import { resolveFactoryEmbedCanvasChrome } from "./factoryEmbedCanvasChrome";

function refOf<T>(value: T): RefObject<T> {
  return { current: value };
}

function buildInput(overrides: Partial<Parameters<typeof resolveFactoryEmbedCanvasChrome>[0]> = {}) {
  const handleSelectLiveCanvas = vi.fn();
  const handleBackToRunList = vi.fn();
  return {
    input: {
      factoryEmbed: true,
      factoryViewOnly: true,
      factoryOwnedApp: false,
      headerBanner: null,
      canUpdateCanvas: false,
      showBottomStatusControls: false,
      hideAddControls: false,
      editSessionActive: false,
      runInspectionChromeActive: true,
      handleSelectMemoryMode: vi.fn(),
      handleSelectConsoleMode: vi.fn(),
      handleSelectFilesMode: vi.fn(),
      handleEnterEditModeFromHeader: vi.fn(),
      handleExitEditSession: vi.fn(),
      handleSelectLiveCanvas,
      handleBackToRunList,
      filesHeaderActionsSlotId: "files-actions",
      runsHasFitToViewRef: refOf(false),
      hasFitToViewRef: refOf(false),
      runsViewportRef: refOf<{ x: number; y: number; zoom: number } | undefined>(undefined),
      viewportRef: refOf<{ x: number; y: number; zoom: number } | undefined>(undefined),
      selectedRunId: "run-1",
      selectedRunDetailDismissed: false,
      runCanvasLoading: false,
      selectedRun: null,
      ...overrides,
    },
    handleSelectLiveCanvas,
    handleBackToRunList,
  };
}

describe("resolveFactoryEmbedCanvasChrome", () => {
  it("keeps run scope on factory close instead of clearing to Live", () => {
    const { input, handleSelectLiveCanvas, handleBackToRunList } = buildInput({ factoryEmbed: true });
    const chrome = resolveFactoryEmbedCanvasChrome(input);

    expect(chrome.onRunNodeDetailClose).toBe(handleBackToRunList);
    expect(chrome.onRunNodeDetailClose).not.toBe(handleSelectLiveCanvas);
    expect(chrome.onBackToLiveCanvas).toBeUndefined();
  });

  it("uses back-to-run-list close outside factory embed", () => {
    const { input, handleBackToRunList, handleSelectLiveCanvas } = buildInput({ factoryEmbed: false });
    const chrome = resolveFactoryEmbedCanvasChrome(input);

    expect(chrome.onRunNodeDetailClose).toBe(handleBackToRunList);
    expect(chrome.onBackToLiveCanvas).toBe(handleSelectLiveCanvas);
  });

  it("hides the canvas tool sidebar in factory embed by default", () => {
    const { input } = buildInput({ factoryEmbed: true });
    const chrome = resolveFactoryEmbedCanvasChrome(input);

    expect(chrome.hideCanvasToolSidebar).toBe(true);
  });

  it("shows the canvas tool sidebar when factory agents are enabled", () => {
    const { input } = buildInput({ factoryEmbed: true, factoryAgentEnabled: true });
    const chrome = resolveFactoryEmbedCanvasChrome(input);

    expect(chrome.hideCanvasToolSidebar).toBe(false);
  });
});
