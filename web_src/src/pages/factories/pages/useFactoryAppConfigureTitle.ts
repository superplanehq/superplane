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
};

export function useFactoryAppConfigureTitle(args: UseFactoryAppConfigureTitleArgs) {
  const { organizationId, factoryId, appId, isConfigure, canRename, savedName, configureBusy, configureActionsRef } =
    args;
  const queryClient = useQueryClient();
  const updateCanvas = useUpdateCanvas(organizationId, appId);
  const [draftTitle, setDraftTitle] = useState<string | null>(null);
  const renamingRef = useRef(false);

  const savedTitle = resolveFactoryAppCanvasTitle(savedName);
  const title = isConfigure && draftTitle != null ? draftTitle : savedTitle;
  const busy = configureBusy || updateCanvas.isPending || renamingRef.current;

  useEffect(() => {
    if (!isConfigure) {
      setDraftTitle(null);
    }
  }, [isConfigure]);

  const clearDraftTitle = useCallback(() => {
    setDraftTitle(null);
  }, []);

  const handleDraftTitleChange = useCallback((name: string) => {
    setDraftTitle(name);
  }, []);

  const persistDraftTitleIfNeeded = useCallback(async () => {
    const nextName = (draftTitle ?? savedTitle).trim();
    if (!nextName || nextName === savedTitle.trim()) {
      return true;
    }
    if (!canRename) {
      showErrorToast("You don't have permission to rename this automation.");
      return false;
    }
    try {
      await updateCanvas.mutateAsync({ name: nextName });
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      setDraftTitle(null);
      return true;
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename automation"));
      return false;
    }
  }, [canRename, draftTitle, factoryId, organizationId, queryClient, savedTitle, updateCanvas]);

  const handleConfigureSave = useCallback(() => {
    if (configureBusy || updateCanvas.isPending) {
      return;
    }
    renamingRef.current = true;
    void (async () => {
      try {
        const renamed = await persistDraftTitleIfNeeded();
        if (!renamed) {
          return;
        }
        configureActionsRef.current?.save();
      } finally {
        renamingRef.current = false;
      }
    })();
  }, [configureActionsRef, configureBusy, persistDraftTitleIfNeeded, updateCanvas.isPending]);

  const handleConfigureDiscard = useCallback(() => {
    if (configureBusy || updateCanvas.isPending) {
      return;
    }
    setDraftTitle(null);
    configureActionsRef.current?.discard();
  }, [configureActionsRef, configureBusy, updateCanvas.isPending]);

  return {
    title,
    configureBusy: busy || updateCanvas.isPending,
    handleDraftTitleChange,
    handleConfigureSave,
    handleConfigureDiscard,
    clearDraftTitle,
  };
}
