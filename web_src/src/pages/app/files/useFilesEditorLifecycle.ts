import { useEffect, type Dispatch, type SetStateAction } from "react";

type UseFilesEditorLifecycleOptions = {
  isEditing: boolean;
  resetPendingState: () => void;
  setIsDiffOpen: (open: boolean) => void;
  headerActionsSlotId?: string;
  setHeaderActionsHost: (host: HTMLElement | null) => void;
  selectedPath: string | null;
  selectedFileData?: { path?: string; content?: string };
  setLoadedContentByPath: Dispatch<SetStateAction<Record<string, string>>>;
};

export function useFilesEditorLifecycle({
  isEditing,
  resetPendingState,
  setIsDiffOpen,
  headerActionsSlotId,
  setHeaderActionsHost,
  selectedPath,
  selectedFileData,
  setLoadedContentByPath,
}: UseFilesEditorLifecycleOptions) {
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
