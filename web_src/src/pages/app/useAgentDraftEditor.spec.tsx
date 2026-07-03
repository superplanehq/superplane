import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast } from "@/lib/toast";
import type { CanvasPageHeaderMode } from "./viewState";
import { useAgentDraftEditor } from "./useAgentDraftEditor";
import { fetchCanvasVersionWithSpec } from "./lib/repository-spec-files";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showInfoToast: vi.fn(),
}));

vi.mock("./lib/repository-spec-files", () => ({
  fetchCanvasVersionWithSpec: vi.fn(),
  canvasVersionExists: vi.fn(),
}));

function makeDraftVersion(versionId: string): CanvasesCanvasVersion {
  return {
    metadata: {
      id: versionId,
      state: "STATE_DRAFT",
    },
    spec: {
      nodes: [],
    },
  } as CanvasesCanvasVersion;
}

function makePublishedVersion(versionId: string): CanvasesCanvasVersion {
  return {
    metadata: {
      id: versionId,
      state: "STATE_PUBLISHED",
    },
    spec: {
      nodes: [],
    },
  } as CanvasesCanvasVersion;
}

function dispatchDraftReady(detail: { versionId?: string; canvasId?: string }) {
  act(() => {
    window.dispatchEvent(new CustomEvent("agent:draft-ready", { detail }));
  });
}

function setupHook({
  canvasId,
  versionId,
  liveCanvasVersionId,
  headerMode = "version-live",
  isRunInspectionMode = false,
  hasEditableVersion = false,
  hasLocalSaveActivity = false,
  activeCanvasVersionId = "live-version",
  activateCanvasVersionForEditing = vi.fn(() => true),
  enterEditSession = vi.fn(),
  selectableVersionsById = new Map<string, CanvasesCanvasVersion>([[versionId, makeDraftVersion(versionId)]]),
}: {
  canvasId: string;
  versionId: string;
  liveCanvasVersionId?: string;
  headerMode?: CanvasPageHeaderMode;
  isRunInspectionMode?: boolean;
  hasEditableVersion?: boolean;
  hasLocalSaveActivity?: boolean;
  activeCanvasVersionId?: string;
  activateCanvasVersionForEditing?: (versionId: string, version: CanvasesCanvasVersion) => boolean;
  enterEditSession?: () => void;
  selectableVersionsById?: Map<string, CanvasesCanvasVersion>;
}) {
  const queryClient = new QueryClient();
  const activeCanvasVersionIdRef = { current: activeCanvasVersionId };
  const onActiveDraftMissing = vi.fn();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  const initialProps = {
    canvasId,
    liveCanvasVersionId,
    headerMode,
    isRunInspectionMode,
    selectableVersionsById,
    hasEditableVersion,
    hasLocalSaveActivity,
    activeCanvasVersionIdRef,
    activateCanvasVersionForEditing,
    enterEditSession,
    onActiveDraftMissing,
  };

  const hook = renderHook((props) => useAgentDraftEditor(props), { initialProps, wrapper });

  return {
    ...hook,
    activeCanvasVersionIdRef,
    activateCanvasVersionForEditing,
    enterEditSession,
    selectableVersionsById,
    onActiveDraftMissing,
    updateProps: (overrides: Partial<typeof initialProps>) => hook.rerender({ ...initialProps, ...overrides }),
  };
}

describe("useAgentDraftEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("retries skipped auto-open when the user returns to the workflow tab", async () => {
    const versionId = "draft-retry-workflow-tab";
    const hook = setupHook({
      canvasId: "canvas-retry-workflow-tab",
      versionId,
      headerMode: "memory",
    });

    dispatchDraftReady({ versionId });

    await act(async () => undefined);
    expect(hook.activateCanvasVersionForEditing).not.toHaveBeenCalled();

    hook.updateProps({ headerMode: "version-live" });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));
    expect(hook.enterEditSession).toHaveBeenCalledTimes(1);
    expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledWith(
      versionId,
      hook.selectableVersionsById.get(versionId),
    );
  });

  it("retries skipped auto-open after run inspection ends", async () => {
    const versionId = "draft-retry-run-inspection";
    const hook = setupHook({
      canvasId: "canvas-retry-run-inspection",
      versionId,
      isRunInspectionMode: true,
    });

    dispatchDraftReady({ versionId });

    await act(async () => undefined);
    expect(hook.activateCanvasVersionForEditing).not.toHaveBeenCalled();

    hook.updateProps({ isRunInspectionMode: false });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));
    expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledWith(
      versionId,
      hook.selectableVersionsById.get(versionId),
    );
  });

  it("retries skipped auto-open after local draft conflicts clear", async () => {
    const versionId = "draft-retry-conflict-clear";
    const hook = setupHook({
      canvasId: "canvas-retry-conflict-clear",
      versionId,
      hasEditableVersion: true,
      hasLocalSaveActivity: true,
      activeCanvasVersionId: "other-draft",
    });

    dispatchDraftReady({ versionId });

    await act(async () => undefined);
    expect(hook.activateCanvasVersionForEditing).not.toHaveBeenCalled();

    hook.updateProps({ hasLocalSaveActivity: false });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));
    expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledWith(
      versionId,
      hook.selectableVersionsById.get(versionId),
    );
  });

  it("does not auto-open the same draft again after a successful auto-open", async () => {
    const versionId = "draft-open-once";
    const hook = setupHook({
      canvasId: "canvas-open-once",
      versionId,
    });

    dispatchDraftReady({ versionId });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));

    dispatchDraftReady({ versionId });

    await act(async () => undefined);
    expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1);
  });

  it("retries auto-open when activation does not apply the draft", async () => {
    const versionId = "draft-activation-skipped";
    const activateCanvasVersionForEditing = vi.fn(() => false);
    const hook = setupHook({
      canvasId: "canvas-activation-skipped",
      versionId,
      activateCanvasVersionForEditing,
    });

    dispatchDraftReady({ versionId });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));

    activateCanvasVersionForEditing.mockReturnValue(true);
    hook.updateProps({ hasLocalSaveActivity: true });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(2));
  });

  it("uses the live version id when staging-actions omit versionId", async () => {
    const liveVersionId = "live-version-fallback";
    const hook = setupHook({
      canvasId: "canvas-live-fallback",
      versionId: liveVersionId,
      liveCanvasVersionId: liveVersionId,
      selectableVersionsById: new Map([[liveVersionId, makePublishedVersion(liveVersionId)]]),
    });

    dispatchDraftReady({ canvasId: "canvas-live-fallback" });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));
    expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledWith(
      liveVersionId,
      hook.selectableVersionsById.get(liveVersionId),
    );
    expect(hook.enterEditSession).toHaveBeenCalledTimes(1);
  });

  it("opens the live version for editing when auto-open loads a published version", async () => {
    const versionId = "live-version-for-staging";
    vi.mocked(fetchCanvasVersionWithSpec).mockResolvedValue(makePublishedVersion(versionId));
    const hook = setupHook({
      canvasId: "canvas-live-for-staging",
      versionId,
      selectableVersionsById: new Map(),
    });

    dispatchDraftReady({ versionId });

    await waitFor(() =>
      expect(fetchCanvasVersionWithSpec).toHaveBeenCalledWith("canvas-live-for-staging", versionId),
    );
    expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledWith(
      versionId,
      makePublishedVersion(versionId),
    );
    expect(hook.enterEditSession).toHaveBeenCalledTimes(1);
    expect(showErrorToast).not.toHaveBeenCalled();
  });
});
