import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

import {
  agentPromptsMatchDefault,
  applyDefaultAgentPrompts,
  RESTORE_DEFAULT_PROMPT_COPY,
} from "../lib/defaultAgentPrompt";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

export function useRestoreDefaultPrompt({
  draft,
  setDraft,
  onRestoreDefaultPrompt,
}: {
  draft: PlanningReviewDraft;
  setDraft: Dispatch<SetStateAction<PlanningReviewDraft>>;
  onRestoreDefaultPrompt?: () => Promise<PlanningReviewStep[] | null>;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [isRestoring, setIsRestoring] = useState(false);
  const [defaultSteps, setDefaultSteps] = useState<PlanningReviewStep[] | null>(null);
  const [restoreError, setRestoreError] = useState<string | undefined>();
  const [applyOnRetry, setApplyOnRetry] = useState(false);
  const restoreLoaderRef = useRef(onRestoreDefaultPrompt);
  restoreLoaderRef.current = onRestoreDefaultPrompt;
  const promptMatchesDefault = defaultSteps != null && agentPromptsMatchDefault(draft, defaultSteps);
  const showRestore = Boolean(onRestoreDefaultPrompt) && !restoreError && !promptMatchesDefault && defaultSteps != null;

  useEffect(() => {
    if (!onRestoreDefaultPrompt) {
      return;
    }
    let cancelled = false;
    void loadDefaultSteps(restoreLoaderRef)
      .then((steps) => {
        if (!cancelled) {
          rememberDefaultSteps(steps, setDefaultSteps, setRestoreError);
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          rememberRestoreError(error, setDefaultSteps, setRestoreError);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [onRestoreDefaultPrompt]);

  const applyDefaultPrompt = async () => {
    if (!restoreLoaderRef.current || isRestoring) {
      return;
    }
    setConfirmOpen(false);
    setIsRestoring(true);
    setRestoreError(undefined);
    try {
      const steps = await loadDefaultSteps(restoreLoaderRef);
      setDefaultSteps(steps);
      setApplyOnRetry(false);
      setDraft((current) => applyDefaultAgentPrompts(current, steps));
    } catch (error) {
      setApplyOnRetry(true);
      const message = rememberRestoreError(error, setDefaultSteps, setRestoreError);
      showErrorToast(message);
    } finally {
      setIsRestoring(false);
    }
  };

  const retryRestore = () => {
    if (isRestoring) {
      return;
    }
    if (applyOnRetry) {
      void applyDefaultPrompt();
      return;
    }
    setIsRestoring(true);
    void loadDefaultSteps(restoreLoaderRef)
      .then((steps) => rememberDefaultSteps(steps, setDefaultSteps, setRestoreError))
      .catch((error: unknown) => rememberRestoreError(error, setDefaultSteps, setRestoreError))
      .finally(() => setIsRestoring(false));
  };

  return {
    confirmOpen,
    setConfirmOpen,
    isRestoring,
    restoreError,
    showRestore,
    applyDefaultPrompt,
    retryRestore,
  };
}

function rememberDefaultSteps(
  steps: PlanningReviewStep[],
  setDefaultSteps: (steps: PlanningReviewStep[]) => void,
  setRestoreError: (error: string | undefined) => void,
) {
  setDefaultSteps(steps);
  setRestoreError(undefined);
}

function rememberRestoreError(
  error: unknown,
  setDefaultSteps: (steps: PlanningReviewStep[] | null) => void,
  setRestoreError: (error: string | undefined) => void,
) {
  const message = getApiErrorMessage(error, RESTORE_DEFAULT_PROMPT_COPY.error);
  setDefaultSteps(null);
  setRestoreError(message);
  return message;
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
