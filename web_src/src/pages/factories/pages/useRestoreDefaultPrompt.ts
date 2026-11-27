import { useCallback, useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

import {
  agentPromptsMatchDefault,
  applyDefaultAgentPrompts,
  RESTORE_DEFAULT_PROMPT_COPY,
} from "../lib/defaultAgentPrompt";
import { loadDefaultAgentPrompt } from "../lib/loadDefaultAgentPrompt";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export function useRestoreDefaultPrompt({
  draft,
  setDraft,
  organizationId,
  factoryId,
  automationId,
  defaultsAvailable,
}: {
  draft: PlanningReviewDraft;
  setDraft: Dispatch<SetStateAction<PlanningReviewDraft>>;
  organizationId?: string;
  factoryId?: string;
  automationId?: string;
  defaultsAvailable?: boolean;
}) {
  const { canAct } = usePermissions();
  const agentNodeId = draft.components[0]?.id;
  const restoreEnabled = Boolean(
    defaultsAvailable && canAct("factories", "update") && organizationId && factoryId && automationId && agentNodeId,
  );
  const loadRestorePrompt = useCallback(() => {
    if (!organizationId || !factoryId || !automationId || !agentNodeId) {
      return Promise.reject(new Error(RESTORE_DEFAULT_PROMPT_COPY.error));
    }
    return loadDefaultAgentPrompt({ organizationId, factoryId, automationId, agentNodeId });
  }, [agentNodeId, automationId, factoryId, organizationId]);
  const onRestoreDefaultPrompt = restoreEnabled ? loadRestorePrompt : undefined;
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [isRestoring, setIsRestoring] = useState(false);
  const [loadedDefaults, setLoadedDefaults] = useState<{
    loader: typeof loadRestorePrompt;
    steps: PlanningReviewStep[];
  } | null>(null);
  const [restoreError, setRestoreError] = useState<string | undefined>();
  const [applyOnRetry, setApplyOnRetry] = useState(false);
  const restoreLoaderRef = useRef(onRestoreDefaultPrompt);
  restoreLoaderRef.current = onRestoreDefaultPrompt;
  const defaultSteps = loadedDefaults && loadedDefaults.loader === onRestoreDefaultPrompt ? loadedDefaults.steps : null;
  const promptMatchesDefault = defaultSteps != null && agentPromptsMatchDefault(draft, defaultSteps);
  const showRestore = Boolean(onRestoreDefaultPrompt) && !restoreError && !promptMatchesDefault && defaultSteps != null;

  useEffect(() => {
    setLoadedDefaults(null);
    setRestoreError(undefined);
    setConfirmOpen(false);
    setApplyOnRetry(false);
    setIsRestoring(false);
    if (!onRestoreDefaultPrompt) {
      return;
    }
    let cancelled = false;
    void loadDefaultSteps(restoreLoaderRef)
      .then((steps) => {
        if (!cancelled) {
          setLoadedDefaults({ loader: onRestoreDefaultPrompt, steps });
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          console.warn("Could not load the default agent prompt.", error);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [onRestoreDefaultPrompt, loadRestorePrompt]);

  const applyDefaultPrompt = async () => {
    if (!restoreLoaderRef.current || isRestoring) {
      return;
    }
    setConfirmOpen(false);
    setIsRestoring(true);
    setRestoreError(undefined);
    const loader = restoreLoaderRef.current;
    try {
      const steps = await loadDefaultSteps(restoreLoaderRef);
      if (restoreLoaderRef.current !== loader) return;
      setLoadedDefaults({ loader, steps });
      setApplyOnRetry(false);
      setDraft((current) => applyDefaultAgentPrompts(current, steps));
    } catch (error) {
      if (restoreLoaderRef.current !== loader) return;
      setApplyOnRetry(true);
      const message = getApiErrorMessage(error, RESTORE_DEFAULT_PROMPT_COPY.error);
      setLoadedDefaults(null);
      setRestoreError(message);
      showErrorToast(message);
    } finally {
      if (restoreLoaderRef.current === loader) setIsRestoring(false);
    }
  };

  const retryRestore = () => {
    if (isRestoring || !applyOnRetry) {
      return;
    }
    void applyDefaultPrompt();
  };

  return {
    confirmOpen: restoreEnabled && confirmOpen,
    setConfirmOpen,
    isRestoring,
    restoreError: restoreEnabled ? restoreError : undefined,
    showRestore,
    applyDefaultPrompt,
    retryRestore,
  };
}

async function loadDefaultSteps(restoreLoaderRef: {
  current: (() => Promise<PlanningReviewStep[] | null>) | undefined;
}): Promise<PlanningReviewStep[]> {
  const loader = restoreLoaderRef.current;
  if (!loader) {
    throw new Error(RESTORE_DEFAULT_PROMPT_COPY.error);
  }
  const steps = await loader();
  if (!steps?.some((step) => step.prompt)) {
    throw new Error(RESTORE_DEFAULT_PROMPT_COPY.error);
  }
  return steps;
}
