import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/lib/canvas-staging";
import { useEffect, type Dispatch, type SetStateAction } from "react";

type UseWorkflowFilesEditorLifecycleOptions = {
  isEditing: boolean;
  resetPendingState: () => void;
  setIsDiffOpen: (open: boolean) => void;
  headerActionsSlotId?: string;
  setHeaderActionsHost: (host: HTMLElement | null) => void;
  selectedPath: string | null;
  selectedFileData?: { path?: string; content?: string };
  setLoadedContentByPath: Dispatch<SetStateAction<Record<string, string>>>;
  branchBaselineFiles?: Record<string, string>;
};

export function useWorkflowFilesEditorLifecycle({
  isEditing,
  resetPendingState,
  setIsDiffOpen,
  headerActionsSlotId,
  setHeaderActionsHost,
  selectedPath,
  selectedFileData,
  setLoadedContentByPath,
  branchBaselineFiles,
}: UseWorkflowFilesEditorLifecycleOptions) {
  useEffect(() => {
    if (!branchBaselineFiles) {
      return;
    }

    setLoadedContentByPath((current) => {
      let next = current;
      for (const path of [CANVAS_YAML_PATH, CONSOLE_YAML_PATH]) {
        const baseline = branchBaselineFiles[path];
        if (baseline === undefined || current[path] === baseline) {
          continue;
        }

        next ??= { ...current };
        next[path] = baseline;
      }

      return next ?? current;
    });
  }, [branchBaselineFiles, setLoadedContentByPath]);
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

  useEffect(() => {
    const path = selectedFileData?.path;
    if (!path || path !== selectedPath) return;

    const content = selectedFileData.content || "";
    setLoadedContentByPath((current) => {
      if (current[path] === content) return current;
      return { ...current, [path]: content };
    });
  }, [selectedFileData, selectedPath, setLoadedContentByPath]);
}
