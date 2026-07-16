import type { ComponentsIntegrationRef } from "@/api-client";
import { buildAutosaveSnapshot } from "./settingsTabValidation";

type RunSettingsTabAutosaveOptions = {
  baselineSnapshot: string;
  clearPendingAutosave: () => void;
  currentNodeName: string;
  flushPendingAutosave: () => void;
  nodeConfiguration: Record<string, unknown>;
  onSave: (
    updatedConfiguration: Record<string, unknown>,
    updatedNodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void | Promise<void>;
  queuePendingAutosave: (snapshot: string) => void;
  savingRef: { current: boolean };
  selectedIntegration?: ComponentsIntegrationRef;
  updateAutosaveBaseline: (snapshot: string) => void;
  validateNow: () => void;
};

export async function runSettingsTabAutosave({
  baselineSnapshot,
  clearPendingAutosave,
  currentNodeName,
  flushPendingAutosave,
  nodeConfiguration,
  onSave,
  queuePendingAutosave,
  savingRef,
  selectedIntegration,
  updateAutosaveBaseline,
  validateNow,
}: RunSettingsTabAutosaveOptions): Promise<void> {
  const snapshot = buildAutosaveSnapshot(nodeConfiguration, currentNodeName, selectedIntegration);
  if (snapshot === baselineSnapshot) {
    clearPendingAutosave();
    return;
  }

  validateNow();
  if (currentNodeName.trim() === "") {
    return;
  }

  if (savingRef.current) {
    queuePendingAutosave(snapshot);
    return;
  }

  const result = onSave(nodeConfiguration, currentNodeName, selectedIntegration);
  if (!(result instanceof Promise)) {
    updateAutosaveBaseline(snapshot);
    flushPendingAutosave();
    return;
  }

  savingRef.current = true;
  try {
    await result;
    updateAutosaveBaseline(snapshot);
  } finally {
    savingRef.current = false;
    flushPendingAutosave();
  }
}

export function shouldRunSettingsTabFallbackAutosave(
  isInteractionDisabled: boolean,
  nodeConfiguration: Record<string, unknown>,
  currentNodeName: string,
  selectedIntegration: ComponentsIntegrationRef | undefined,
  baselineSnapshot: string,
): boolean {
  if (isInteractionDisabled || currentNodeName.trim() === "") {
    return false;
  }

  return buildAutosaveSnapshot(nodeConfiguration, currentNodeName, selectedIntegration) !== baselineSnapshot;
}
