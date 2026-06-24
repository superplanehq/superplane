import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast, showInfoToast } from "@/lib/toast";
import { draftVersionId } from "@/lib/draftVersion";
import { canvasKeys } from "@/hooks/useCanvasData";
import type { CanvasPageHeaderMode } from "./viewState";
import { isCanvasWorkflowTab } from "./viewState";
import { isDraftVersion } from "./lib/canvas-versions";
import { canvasVersionExists, fetchCanvasVersionWithSpec } from "./lib/repository-spec-files";
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
  hasLocalSaveActivity: boolean;
  activeCanvasVersionIdRef: { current: string };
  activateCanvasVersionForEditing: (versionId: string, version: CanvasesCanvasVersion) => boolean;
  setSuppressUnpublishedDraftDiscard: (value: boolean) => void;
  // Full recovery (clears state + ref, exits to live) when the draft the user is
  // actively editing turns out to be deleted.
  onActiveDraftMissing: (versionId: string) => void | Promise<void>;
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
        // fetchCanvasVersionWithSpec loads the version and its canvas.yaml in
        // parallel, so a 404 may come from the file, not a deleted version.
        // Only treat it as a missing draft once the version itself is gone.
        if (isNotFoundError(error) && !(await canvasVersionExists(canvasId, versionId))) {
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
  hasLocalSaveActivity,
  activeCanvasVersionIdRef,
  activateCanvasVersionForEditing,
  setSuppressUnpublishedDraftDiscard,
  onActiveDraftMissing,
}: UseAgentDraftEditorArgs) {
  const [pendingAutoOpenVersionId, setPendingAutoOpenVersionId] = useState<string | null>(null);
  const loadAgentDraftVersion = useLoadAgentDraftVersion(canvasId, selectableVersionsById);

  const handleMissingDraft = useCallback(
    (versionId: string, source: AgentDraftOpenSource, notFound: boolean): AgentDraftOpenResult => {
      // A transient load failure already toasted; don't clear state unless the
      // draft is actually gone.
      if (!notFound) {
        return "unavailable";
      }
      // If the deleted draft is the one being edited, run full recovery — just
      // clearing the ref would be undone by the ref<-state sync effect.
      if (activeCanvasVersionIdRef.current === versionId) {
        void onActiveDraftMissing(versionId);
        return "unavailable";
      }
      if (source === "button") {
        showInfoToast("This draft no longer exists.");
      }
      return "unavailable";
    },
    [activeCanvasVersionIdRef, onActiveDraftMissing],
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

      if (hasEditableVersion && hasLocalSaveActivity && activeCanvasVersionIdRef.current !== versionId) {
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
        return handleMissingDraft(versionId, source, notFound);
      }

      if (!isDraftVersion(version)) {
        if (source === "button") {
          showErrorToast("Agent draft is no longer available");
        }
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
      hasLocalSaveActivity,
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
