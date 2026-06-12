import { useCallback, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast } from "@/lib/toast";
import { draftVersionId } from "@/lib/draftVersion";
import { canvasKeys } from "@/hooks/useCanvasData";
import type { CanvasPageHeaderMode } from "./viewState";
import { isCanvasWorkflowTab } from "./viewState";
import { isDraftVersion } from "./lib/canvas-versions";
import { fetchCanvasVersionWithSpec } from "./lib/repository-spec-files";

type AgentDraftOpenSource = "auto" | "button";

type UseAgentDraftEditorArgs = {
  canvasId: string;
  headerMode: CanvasPageHeaderMode;
  selectableVersionsById: Map<string, CanvasesCanvasVersion>;
  hasEditableVersion: boolean;
  hasPendingLocalCanvasState: boolean;
  activeCanvasVersionIdRef: { current: string };
  activateCanvasVersionForEditing: (versionId: string, version: CanvasesCanvasVersion) => void;
  setSuppressUnpublishedDraftDiscard: (value: boolean) => void;
};

export function useAgentDraftEditor({
  canvasId,
  headerMode,
  selectableVersionsById,
  hasEditableVersion,
  hasPendingLocalCanvasState,
  activeCanvasVersionIdRef,
  activateCanvasVersionForEditing,
  setSuppressUnpublishedDraftDiscard,
}: UseAgentDraftEditorArgs) {
  const queryClient = useQueryClient();

  const loadAgentDraftVersion = useCallback(
    async (versionId: string): Promise<CanvasesCanvasVersion | null> => {
      const cachedVersion = selectableVersionsById.get(versionId);
      if (cachedVersion) {
        return cachedVersion;
      }

      if (!canvasId) {
        return null;
      }

      try {
        const loadedVersion = await fetchCanvasVersionWithSpec(canvasId, versionId);
        if (!loadedVersion?.metadata?.id) {
          return null;
        }

        queryClient.setQueryData<CanvasesCanvasVersion>(canvasKeys.versionDetail(canvasId, versionId), loadedVersion);
        queryClient.setQueryData<CanvasesCanvasVersion[]>(canvasKeys.versionList(canvasId), (current = []) => {
          if (current.some((version) => version.metadata?.id === versionId)) {
            return current;
          }
          return [loadedVersion, ...current];
        });

        if (isDraftVersion(loadedVersion)) {
          queryClient.setQueryData<CanvasesCanvasVersion[]>(canvasKeys.draftBranches(canvasId), (current = []) => {
            if (current.some((branch) => draftVersionId(branch) === versionId)) {
              return current;
            }
            return [loadedVersion, ...current];
          });
        }

        return loadedVersion;
      } catch {
        showErrorToast("Failed to load draft version");
        return null;
      }
    },
    [canvasId, queryClient, selectableVersionsById],
  );

  const openAgentDraftVersion = useCallback(
    async (versionId: string, source: AgentDraftOpenSource) => {
      if (!versionId) {
        return;
      }

      if (source === "auto" && !isCanvasWorkflowTab(headerMode)) {
        return;
      }

      if (activeCanvasVersionIdRef.current === versionId && hasEditableVersion) {
        return;
      }

      if (hasEditableVersion && hasPendingLocalCanvasState && activeCanvasVersionIdRef.current !== versionId) {
        if (source === "auto") {
          return;
        }

        const shouldSwitch = window.confirm(
          "You have unsaved changes in the current draft. Switch to the agent draft and discard those unsaved changes?",
        );
        if (!shouldSwitch) {
          return;
        }
      }

      const version = await loadAgentDraftVersion(versionId);
      if (!version) {
        showErrorToast("Draft version not found");
        return;
      }

      if (!isDraftVersion(version)) {
        showErrorToast("Agent draft is no longer available");
        return;
      }

      setSuppressUnpublishedDraftDiscard(false);
      activateCanvasVersionForEditing(versionId, version);
    },
    [
      activateCanvasVersionForEditing,
      activeCanvasVersionIdRef,
      hasEditableVersion,
      hasPendingLocalCanvasState,
      headerMode,
      loadAgentDraftVersion,
      setSuppressUnpublishedDraftDiscard,
    ],
  );

  useEffect(() => {
    const handleViewVersion = (event: Event) => {
      const versionId = (event as CustomEvent<{ versionId?: string }>).detail?.versionId;
      if (!versionId) return;
      void openAgentDraftVersion(versionId, "button");
    };

    const handleDraftReady = (event: Event) => {
      const versionId = (event as CustomEvent<{ versionId?: string }>).detail?.versionId;
      if (!versionId) return;
      void openAgentDraftVersion(versionId, "auto");
    };

    window.addEventListener("agent:view-version", handleViewVersion);
    window.addEventListener("agent:draft-ready", handleDraftReady);
    return () => {
      window.removeEventListener("agent:view-version", handleViewVersion);
      window.removeEventListener("agent:draft-ready", handleDraftReady);
    };
  }, [openAgentDraftVersion]);
}
