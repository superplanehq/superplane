import { useUpdateCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { FactoryConfigureActions } from "@/pages/app";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState, type MutableRefObject } from "react";
import { resolveFactoryAppCanvasTitle } from "../lib/factoryAppCanvasCopy";

type UseFactoryAppConfigureTitleArgs = {
  organizationId: string;
  factoryId: string;
  appId: string;
  isConfigure: boolean;
  canRename: boolean;
  savedName?: string;
  configureBusy: boolean;
  configureActionsRef: MutableRefObject<FactoryConfigureActions | null>;
  onDone: () => void;
};

export function resolveDraftTitleToPersist(draftTitle: string | null, savedTitle: string): string | null {
  const nextName = (draftTitle ?? savedTitle).trim();
  if (!nextName || nextName === savedTitle.trim()) {
    return null;
  }
  return nextName;
}

export function useFactoryAppConfigureTitle(args: UseFactoryAppConfigureTitleArgs) {
  const {
    organizationId,
    factoryId,
    appId,
    isConfigure,
    canRename,
    savedName,
    configureBusy,
    configureActionsRef,
    onDone,
  } = args;
  const queryClient = useQueryClient();
  const updateCanvas = useUpdateCanvas(organizationId, appId);
  const [draftTitle, setDraftTitle] = useState<string | null>(null);
  const draftTitleRef = useRef<string | null>(null);
  const savingRef = useRef(false);
  const onDoneRef = useRef(onDone);
  onDoneRef.current = onDone;

  const savedTitle = resolveFactoryAppCanvasTitle(savedName);
  const title = isConfigure && draftTitle != null ? draftTitle : savedTitle;

  useEffect(() => {
    if (!isConfigure) {
      draftTitleRef.current = null;
      setDraftTitle(null);
    }
  }, [isConfigure]);

  const clearDraftTitle = useCallback(() => {
    draftTitleRef.current = null;
    setDraftTitle(null);
  }, []);

  const handleDraftTitleChange = useCallback((name: string) => {
    draftTitleRef.current = name;
    setDraftTitle(name);
  }, []);

  const persistDraftTitleIfNeeded = useCallback(async (): Promise<string | null> => {
    const nextName = resolveDraftTitleToPersist(draftTitleRef.current, savedTitle);
    if (!nextName) {
      return null;
    }
    if (!canRename) {
      showErrorToast("You don't have permission to rename this automation.");
      throw new Error("rename not allowed");
    }
    await updateCanvas.mutateAsync({ name: nextName });
    void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    draftTitleRef.current = null;
    setDraftTitle(null);
    return nextName;
  }, [canRename, factoryId, organizationId, queryClient, savedTitle, updateCanvas]);

  const handleConfigureSave = useCallback(() => {
    if (configureBusy || updateCanvas.isPending || savingRef.current) {
      return;
    }
    savingRef.current = true;
    void (async () => {
      try {
        let persistedName: string | null = null;
        try {
          persistedName = await persistDraftTitleIfNeeded();
        } catch (error) {
          if (error instanceof Error && error.message === "rename not allowed") {
            return;
          }
          showErrorToast(getApiErrorMessage(error, "Failed to rename automation"));
          return;
        }

        const actions = configureActionsRef.current;
        if (!actions) {
          onDoneRef.current();
          return;
        }

        // Always stage through Configure Save. Trusting hasUncommittedChanges alone
        // can skip real graph edits while staging baselines are still loading.
        // Empty staging (no YAML diff) finishes cleanly inside runFactoryConfigureSave.
        actions.save({ canvasName: persistedName ?? undefined });
      } finally {
        savingRef.current = false;
      }
    })();
  }, [configureActionsRef, configureBusy, persistDraftTitleIfNeeded, updateCanvas.isPending]);

  const handleConfigureDiscard = useCallback(() => {
    if (configureBusy || updateCanvas.isPending) {
      return;
    }
    clearDraftTitle();
    configureActionsRef.current?.discard();
  }, [clearDraftTitle, configureActionsRef, configureBusy, updateCanvas.isPending]);

  return {
    title,
    configureBusy: configureBusy || updateCanvas.isPending,
    handleDraftTitleChange,
    handleConfigureSave,
    handleConfigureDiscard,
    clearDraftTitle,
  };
}
