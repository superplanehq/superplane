import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast } from "@/lib/toast";
import { draftVersionId } from "@/lib/draftVersion";
import { canvasKeys } from "@/hooks/useCanvasData";
import type { CanvasPageHeaderMode } from "./viewState";
import { isCanvasWorkflowTab } from "./viewState";
import { isDraftVersion } from "./lib/canvas-versions";
import { fetchCanvasVersionWithSpec } from "./lib/repository-spec-files";
import { isNotFoundError } from "./workflowPageHelpers";

type AgentDraftOpenSource = "auto" | "button";
type AgentDraftOpenResult = "opened" | "skipped" | "unavailable";
// `notFound` distinguishes a genuinely-deleted draft from a transient load
// failure, so only the former clears the active ref.
type LoadAgentDraftResult = { version: CanvasesCanvasVersion | null; notFound: boolean };
type LoadAgentDraftVersion = (versionId: string) => Promise<LoadAgentDraftResult>;

const autoOpenedAgentDraftKeys = new Set<string>();

function agentDraftAutoOpenKey(canvasId: string, versionId: string): string {
  return `${canvasId}:${versionId}`;
}

type UseAgentDraftEditorArgs = {
  canvasId: string;
  headerMode: CanvasPageHeaderMode;
  isRunInspectionMode: boolean;
  selectableVersionsById: Map<string, CanvasesCanvasVersion>;
  hasEditableVersion: boolean;
  hasPendingLocalCanvasState: boolean;
  activeCanvasVersionIdRef: { current: string };
  activateCanvasVersionForEditing: (versionId: string, version: CanvasesCanvasVersion) => boolean;
  setSuppressUnpublishedDraftDiscard: (value: boolean) => void;
};

function useLoadAgentDraftVersion(
  canvasId: string,
  selectableVersionsById: Map<string, CanvasesCanvasVersion>,
): LoadAgentDraftVersion {
  const queryClient = useQueryClient();

  // Drop a deleted draft id from the cached lists so it stops being offered.
  const pruneDeadDraft = useCallback(
    (versionId: string) => {
      if (!canvasId) {
        return;
      }
      queryClient.setQueryData<CanvasesCanvasVersion[]>(canvasKeys.draftBranches(canvasId), (current = []) =>
        current.filter((branch) => draftVersionId(branch) !== versionId),
      );
      queryClient.setQueryData<CanvasesCanvasVersion[]>(canvasKeys.versionList(canvasId), (current = []) =>
        current.filter((version) => version.metadata?.id !== versionId),
      );
    },
    [canvasId, queryClient],
  );

  return useCallback(
    async (versionId: string): Promise<LoadAgentDraftResult> => {
      const cachedVersion = selectableVersionsById.get(versionId);
      if (cachedVersion) {
        return { version: cachedVersion, notFound: false };
      }

      if (!canvasId) {
        return { version: null, notFound: false };
      }

      try {
        const loadedVersion = await fetchCanvasVersionWithSpec(canvasId, versionId);
        if (!loadedVersion?.metadata?.id) {
          pruneDeadDraft(versionId);
          return { version: null, notFound: true };
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

        return { version: loadedVersion, notFound: false };
      } catch (error) {
        if (isNotFoundError(error)) {
          pruneDeadDraft(versionId);
          return { version: null, notFound: true };
        }
        showErrorToast("Failed to load draft version");
        return { version: null, notFound: false };
      }
    },
    [canvasId, pruneDeadDraft, queryClient, selectableVersionsById],
  );
}

function usePendingAgentDraftAutoOpen({
  canvasId,
  pendingAutoOpenVersionId,
  setPendingAutoOpenVersionId,
  openAgentDraftVersion,
}: {
  canvasId: string;
  pendingAutoOpenVersionId: string | null;
  setPendingAutoOpenVersionId: Dispatch<SetStateAction<string | null>>;
  openAgentDraftVersion: (versionId: string, source: "auto") => Promise<AgentDraftOpenResult>;
}) {
  useEffect(() => {
    if (!pendingAutoOpenVersionId || !canvasId) {
      return;
    }

    const key = agentDraftAutoOpenKey(canvasId, pendingAutoOpenVersionId);
    if (autoOpenedAgentDraftKeys.has(key)) {
      setPendingAutoOpenVersionId(null);
      return;
    }

    let cancelled = false;

    void openAgentDraftVersion(pendingAutoOpenVersionId, "auto").then((result) => {
      if (cancelled) {
        return;
      }

      if (result === "opened") {
        autoOpenedAgentDraftKeys.add(key);
        setPendingAutoOpenVersionId(null);
        return;
      }

      if (result === "unavailable") {
        setPendingAutoOpenVersionId(null);
      }
    });

    return () => {
      cancelled = true;
    };
  }, [canvasId, openAgentDraftVersion, pendingAutoOpenVersionId, setPendingAutoOpenVersionId]);
}

export function useAgentDraftEditor({
  canvasId,
  headerMode,
  isRunInspectionMode,
  selectableVersionsById,
  hasEditableVersion,
  hasPendingLocalCanvasState,
  activeCanvasVersionIdRef,
  activateCanvasVersionForEditing,
  setSuppressUnpublishedDraftDiscard,
}: UseAgentDraftEditorArgs) {
  const [pendingAutoOpenVersionId, setPendingAutoOpenVersionId] = useState<string | null>(null);
  const loadAgentDraftVersion = useLoadAgentDraftVersion(canvasId, selectableVersionsById);

  // Drop the stale ref for a deleted draft so it can't be re-published.
  const handleMissingDraft = useCallback(
    (versionId: string, source: AgentDraftOpenSource): AgentDraftOpenResult => {
      if (activeCanvasVersionIdRef.current === versionId) {
        activeCanvasVersionIdRef.current = "";
      }
      if (source === "button") {
        showErrorToast("This draft no longer exists.");
      }
      return "unavailable";
    },
    [activeCanvasVersionIdRef],
  );

  const openAgentDraftVersion = useCallback(
    async (versionId: string, source: AgentDraftOpenSource): Promise<AgentDraftOpenResult> => {
      if (!versionId) {
        return "unavailable";
      }

      if (source === "auto" && (!isCanvasWorkflowTab(headerMode) || isRunInspectionMode)) {
        return "skipped";
      }

      if (activeCanvasVersionIdRef.current === versionId && hasEditableVersion) {
        return "opened";
      }

      if (hasEditableVersion && hasPendingLocalCanvasState && activeCanvasVersionIdRef.current !== versionId) {
        if (source === "auto") {
          return "skipped";
        }

        const shouldSwitch = window.confirm(
          "You have unsaved changes in the current draft. Switch to the agent draft and discard those unsaved changes?",
        );
        if (!shouldSwitch) {
          return "skipped";
        }
      }

      const { version, notFound } = await loadAgentDraftVersion(versionId);
      if (!version) {
        // A transient load failure already toasted; don't clear the active ref
        // or claim the draft is gone unless it actually is.
        return notFound ? handleMissingDraft(versionId, source) : "unavailable";
      }

      if (!isDraftVersion(version)) {
        showErrorToast("Agent draft is no longer available");
        return "unavailable";
      }

      setSuppressUnpublishedDraftDiscard(false);
      return activateCanvasVersionForEditing(versionId, version) ? "opened" : "skipped";
    },
    [
      activateCanvasVersionForEditing,
      activeCanvasVersionIdRef,
      handleMissingDraft,
      hasEditableVersion,
      hasPendingLocalCanvasState,
      headerMode,
      isRunInspectionMode,
      loadAgentDraftVersion,
      setSuppressUnpublishedDraftDiscard,
    ],
  );

  usePendingAgentDraftAutoOpen({
    canvasId,
    pendingAutoOpenVersionId,
    setPendingAutoOpenVersionId,
    openAgentDraftVersion,
  });

  useEffect(() => {
    const handleViewVersion = (event: Event) => {
      const versionId = (event as CustomEvent<{ versionId?: string }>).detail?.versionId;
      if (!versionId) return;
      void openAgentDraftVersion(versionId, "button");
    };

    const handleDraftReady = (event: Event) => {
      const versionId = (event as CustomEvent<{ versionId?: string }>).detail?.versionId;
      if (!versionId) return;
      if (!canvasId) return;
      if (autoOpenedAgentDraftKeys.has(agentDraftAutoOpenKey(canvasId, versionId))) return;
      setPendingAutoOpenVersionId(versionId);
    };

    window.addEventListener("agent:view-version", handleViewVersion);
    window.addEventListener("agent:draft-ready", handleDraftReady);
    return () => {
      window.removeEventListener("agent:view-version", handleViewVersion);
      window.removeEventListener("agent:draft-ready", handleDraftReady);
    };
  }, [canvasId, openAgentDraftVersion]);
}
