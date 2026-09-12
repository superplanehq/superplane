import { useCallback, useEffect, useState } from "react";

import type { WorkspaceNextStepId } from "./workspaceNextStepCatalog";
import { runWorkspaceNextStepTransition } from "./workspaceNextStepTransition";

export const WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY = "sp:workspace-next-steps-deferred:v1";

type DeferredByFactory = Record<string, WorkspaceNextStepId>;

function isWorkspaceNextStepId(value: unknown): value is WorkspaceNextStepId {
  return value === "pr-comments-handler" || value === "pr-checks-handler";
}

function readStore(): DeferredByFactory {
  try {
    const raw = window.localStorage.getItem(WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return {};
    }
    const store: DeferredByFactory = {};
    for (const [factoryId, stepId] of Object.entries(parsed)) {
      if (factoryId && isWorkspaceNextStepId(stepId)) {
        store[factoryId] = stepId;
      }
    }
    return store;
  } catch {
    return {};
  }
}

function writeStore(store: DeferredByFactory): void {
  try {
    if (Object.keys(store).length === 0) {
      window.localStorage.removeItem(WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY, JSON.stringify(store));
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

export function readDeferredWorkspaceNextStep(factoryId: string): WorkspaceNextStepId | null {
  if (!factoryId) {
    return null;
  }
  return readStore()[factoryId] ?? null;
}

export function writeDeferredWorkspaceNextStep(factoryId: string, stepId: WorkspaceNextStepId | null): void {
  if (!factoryId) {
    return;
  }
  const store = readStore();
  if (stepId) {
    store[factoryId] = stepId;
  } else {
    delete store[factoryId];
  }
  writeStore(store);
}

export function useWorkspaceNextStepDeferral(factoryId: string): {
  deferredStepId: WorkspaceNextStepId | null;
  defer: (stepId: WorkspaceNextStepId) => void;
  restore: () => void;
  forget: () => void;
} {
  const [deferredStepId, setDeferredStepId] = useState<WorkspaceNextStepId | null>(() =>
    readDeferredWorkspaceNextStep(factoryId),
  );

  useEffect(() => {
    setDeferredStepId(readDeferredWorkspaceNextStep(factoryId));
  }, [factoryId]);

  const persist = useCallback(
    (stepId: WorkspaceNextStepId | null) => {
      setDeferredStepId(stepId);
      writeDeferredWorkspaceNextStep(factoryId, stepId);
    },
    [factoryId],
  );

  const defer = useCallback(
    (stepId: WorkspaceNextStepId) => {
      runWorkspaceNextStepTransition(() => {
        persist(stepId);
      });
    },
    [persist],
  );

  const restore = useCallback(() => {
    runWorkspaceNextStepTransition(() => {
      persist(null);
    });
  }, [persist]);

  const forget = useCallback(() => {
    persist(null);
  }, [persist]);

  return { deferredStepId, defer, restore, forget };
}
