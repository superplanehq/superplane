import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";

import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

import { applyDefaultRefinementPrompt, refinementPromptMatchesDefault } from "../lib/refinementPrompt";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";
import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";

export function useRestoreDefaultPrompt({
  draft,
  setDraft,
  onRestoreDefaultPrompt,
}: {
  draft: PlanningReviewDraft;
  setDraft: Dispatch<SetStateAction<PlanningReviewDraft>>;
  onRestoreDefaultPrompt?: () => Promise<PlanningReviewStep | null>;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [isRestoring, setIsRestoring] = useState(false);
  const [defaultStep, setDefaultStep] = useState<PlanningReviewStep | null>(null);
  const [restoreError, setRestoreError] = useState<string | undefined>();
  const [applyOnRetry, setApplyOnRetry] = useState(false);
  const restoreLoaderRef = useRef(onRestoreDefaultPrompt);
  restoreLoaderRef.current = onRestoreDefaultPrompt;
  const promptMatchesDefault = defaultStep != null && refinementPromptMatchesDefault(draft, defaultStep);
  const showRestore = Boolean(onRestoreDefaultPrompt) && !restoreError && !promptMatchesDefault && defaultStep != null;

  useEffect(() => {
    if (!onRestoreDefaultPrompt) {
      return;
    }
    let cancelled = false;
    void loadDefaultStep(restoreLoaderRef)
      .then((step) => {
        if (!cancelled) {
          rememberDefaultStep(step, setDefaultStep, setRestoreError);
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          rememberRestoreError(error, setDefaultStep, setRestoreError);
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
      const step = await loadDefaultStep(restoreLoaderRef);
      setDefaultStep(step);
      setApplyOnRetry(false);
      setDraft((current) => applyDefaultRefinementPrompt(current, step));
    } catch (error) {
      setApplyOnRetry(true);
      const message = rememberRestoreError(error, setDefaultStep, setRestoreError);
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
    void loadDefaultStep(restoreLoaderRef)
      .then((step) => rememberDefaultStep(step, setDefaultStep, setRestoreError))
      .catch((error: unknown) => rememberRestoreError(error, setDefaultStep, setRestoreError))
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

function rememberDefaultStep(
  step: PlanningReviewStep,
  setDefaultStep: (step: PlanningReviewStep) => void,
  setRestoreError: (error: string | undefined) => void,
) {
  setDefaultStep(step);
  setRestoreError(undefined);
}

function rememberRestoreError(
  error: unknown,
  setDefaultStep: (step: PlanningReviewStep | null) => void,
  setRestoreError: (error: string | undefined) => void,
) {
  const message = getApiErrorMessage(error, PLANNING_SETTINGS_COPY.restorePromptError);
  setDefaultStep(null);
  setRestoreError(message);
  return message;
}

async function loadDefaultStep(restoreLoaderRef: {
  current: (() => Promise<PlanningReviewStep | null>) | undefined;
}): Promise<PlanningReviewStep> {
  const loader = restoreLoaderRef.current;
  if (!loader) {
    throw new Error(PLANNING_SETTINGS_COPY.restorePromptError);
  }
  const step = await loader();
  if (!step?.prompt) {
    throw new Error(PLANNING_SETTINGS_COPY.restorePromptError);
  }
  return step;
}
