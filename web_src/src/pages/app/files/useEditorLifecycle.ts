import { useLayoutEffect, useEffect, useRef, type Dispatch, type SetStateAction } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { isWorkflowSpecPath } from "../lib/workflow-spec-paths";
import { fetchRepositoryFileContentCached } from "@/hooks/useCanvasData";

type UseEditorLifecycleOptions = {
  canvasId?: string;
  versionId?: string;
  isEditing: boolean;
  resetPendingState: () => void;
  setIsDiffOpen: (open: boolean) => void;
  headerActionsSlotId?: string;
  setHeaderActionsHost: (host: HTMLElement | null) => void;
  selectedPath: string | null;
  selectedFileData?: { path?: string; content?: string };
  setLoadedContentByPath: Dispatch<SetStateAction<Record<string, string>>>;
  setCommittedContentByPath: Dispatch<SetStateAction<Record<string, string>>>;
  stagingResetNonce?: number;
};

export function useEditorLifecycle({
  canvasId,
  versionId,
  isEditing,
  resetPendingState,
  setIsDiffOpen,
  headerActionsSlotId,
  setHeaderActionsHost,
  selectedPath,
  selectedFileData,
  setLoadedContentByPath,
  setCommittedContentByPath,
  stagingResetNonce = 0,
}: UseEditorLifecycleOptions) {
  const queryClient = useQueryClient();
  const previousStagingResetNonceRef = useRef(stagingResetNonce);

  useEffect(() => {
    if (previousStagingResetNonceRef.current === stagingResetNonce) {
      return;
    }

    previousStagingResetNonceRef.current = stagingResetNonce;
    resetPendingState();
    setLoadedContentByPath({});
    setCommittedContentByPath({});
    setIsDiffOpen(false);
  }, [stagingResetNonce, resetPendingState, setCommittedContentByPath, setIsDiffOpen, setLoadedContentByPath]);

  useEffect(() => {
    if (isEditing) return;

    resetPendingState();
    setIsDiffOpen(false);
  }, [isEditing, resetPendingState, setIsDiffOpen]);

  useEffect(() => {
    if (!headerActionsSlotId || !isEditing) {
      setHeaderActionsHost(null);
      return;
    }

    setHeaderActionsHost(document.getElementById(headerActionsSlotId));
  }, [headerActionsSlotId, isEditing, setHeaderActionsHost]);

  useLayoutEffect(() => {
    const path = selectedFileData?.path;
    if (!path || path !== selectedPath) return;

    const content = selectedFileData.content || "";
    setLoadedContentByPath((current) => {
      if (current[path] === content) return current;
      return { ...current, [path]: content };
    });
  }, [selectedFileData, selectedPath, setLoadedContentByPath]);

  useEffect(() => {
    const path = selectedPath;
    if (!path || !canvasId || !versionId || isWorkflowSpecPath(path)) {
      return;
    }

    let cancelled = false;
    void fetchRepositoryFileContentCached(queryClient, canvasId, path, versionId, false).then((content) => {
      if (cancelled) {
        return;
      }

      setCommittedContentByPath((current) => {
        if (current[path] === content) {
          return current;
        }

        return { ...current, [path]: content };
      });
    });

    return () => {
      cancelled = true;
    };
  }, [canvasId, queryClient, selectedPath, setCommittedContentByPath, versionId]);
}
