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

function dispatchDraftReady(versionId: string) {
  act(() => {
    window.dispatchEvent(new CustomEvent("agent:draft-ready", { detail: { versionId } }));
  });
}

function setupHook({
  canvasId,
  versionId,
  headerMode = "version-live",
  isRunInspectionMode = false,
  hasEditableVersion = false,
  hasLocalSaveActivity = false,
  activeCanvasVersionId = "live-version",
  activateCanvasVersionForEditing = vi.fn(() => true),
  selectableVersionsById = new Map<string, CanvasesCanvasVersion>([[versionId, makeDraftVersion(versionId)]]),
}: {
  canvasId: string;
  versionId: string;
  headerMode?: CanvasPageHeaderMode;
  isRunInspectionMode?: boolean;
  hasEditableVersion?: boolean;
  hasLocalSaveActivity?: boolean;
  activeCanvasVersionId?: string;
  activateCanvasVersionForEditing?: (versionId: string, version: CanvasesCanvasVersion) => boolean;
  selectableVersionsById?: Map<string, CanvasesCanvasVersion>;
}) {
  const queryClient = new QueryClient();
  const activeCanvasVersionIdRef = { current: activeCanvasVersionId };
  const setSuppressUnpublishedDraftDiscard = vi.fn();
  const onActiveDraftMissing = vi.fn();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  const initialProps = {
    canvasId,
    headerMode,
    isRunInspectionMode,
    selectableVersionsById,
    hasEditableVersion,
    hasLocalSaveActivity,
    activeCanvasVersionIdRef,
    activateCanvasVersionForEditing,
    setSuppressUnpublishedDraftDiscard,
    onActiveDraftMissing,
  };

  const hook = renderHook((props) => useAgentDraftEditor(props), { initialProps, wrapper });

  return {
    ...hook,
    activeCanvasVersionIdRef,
    activateCanvasVersionForEditing,
    selectableVersionsById,
    setSuppressUnpublishedDraftDiscard,
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

    dispatchDraftReady(versionId);

    await act(async () => undefined);
    expect(hook.activateCanvasVersionForEditing).not.toHaveBeenCalled();

    hook.updateProps({ headerMode: "version-live" });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));
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

    dispatchDraftReady(versionId);

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

    dispatchDraftReady(versionId);

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

    dispatchDraftReady(versionId);

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));

    dispatchDraftReady(versionId);

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

    dispatchDraftReady(versionId);

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(1));

    activateCanvasVersionForEditing.mockReturnValue(true);
    hook.updateProps({ hasLocalSaveActivity: true });

    await waitFor(() => expect(hook.activateCanvasVersionForEditing).toHaveBeenCalledTimes(2));
  });

  it("does not show an error when auto-open finds the agent draft was already published", async () => {
    const versionId = "draft-published-before-auto-open";
    vi.mocked(fetchCanvasVersionWithSpec).mockResolvedValue(makePublishedVersion(versionId));
    const hook = setupHook({
      canvasId: "canvas-published-before-auto-open",
      versionId,
      selectableVersionsById: new Map(),
    });

    dispatchDraftReady(versionId);

    await waitFor(() =>
      expect(fetchCanvasVersionWithSpec).toHaveBeenCalledWith("canvas-published-before-auto-open", versionId),
    );
    expect(hook.activateCanvasVersionForEditing).not.toHaveBeenCalled();
    expect(showErrorToast).not.toHaveBeenCalledWith("Agent draft is no longer available");
  });
});
