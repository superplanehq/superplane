import type { ComponentsIntegrationRef } from "@/api-client";
import { useCallback, useEffect, useRef } from "react";
import { runSettingsTabAutosave, shouldRunSettingsTabFallbackAutosave } from "./settingsTabAutosaveActions";
import { buildAutosaveSnapshot } from "./settingsTabValidation";

type UseSettingsTabAutosaveOptions = {
  currentNodeName: string;
  initialConfiguration: Record<string, unknown>;
  initialIntegrationRef?: ComponentsIntegrationRef;
  initialNodeName: string;
  isInteractionDisabled: boolean;
  nodeConfiguration: Record<string, unknown>;
  onSave: (
    updatedConfiguration: Record<string, unknown>,
    updatedNodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void | Promise<void>;
  selectedIntegration?: ComponentsIntegrationRef;
  validateNow: () => void;
};

export function useSettingsTabAutosave({
  currentNodeName,
  initialConfiguration,
  initialIntegrationRef,
  initialNodeName,
  isInteractionDisabled,
  nodeConfiguration,
  onSave,
  selectedIntegration,
  validateNow,
}: UseSettingsTabAutosaveOptions) {
  const savingRef = useRef(false);
  const autosaveTimerRef = useRef<number | null>(null);
  const autosaveBaselineSnapshotRef = useRef(
    buildAutosaveSnapshot(initialConfiguration, initialNodeName, initialIntegrationRef),
  );
  const pendingAutosaveSnapshotRef = useRef<string | null>(null);

  const setAutosaveBaseline = useCallback(
    (configuration: Record<string, unknown>, nodeName: string, integrationRef?: ComponentsIntegrationRef) => {
      autosaveBaselineSnapshotRef.current = buildAutosaveSnapshot(configuration, nodeName, integrationRef);
      pendingAutosaveSnapshotRef.current = null;
    },
    [],
  );

  const updateAutosaveBaseline = useCallback((snapshot: string) => {
    autosaveBaselineSnapshotRef.current = snapshot;
    pendingAutosaveSnapshotRef.current = null;
  }, []);

  const queuePendingAutosave = useCallback((snapshot: string) => {
    pendingAutosaveSnapshotRef.current = snapshot;
  }, []);

  const flushPendingAutosave = useCallback(() => {
    const pendingSnapshot = pendingAutosaveSnapshotRef.current;
    if (!pendingSnapshot || pendingSnapshot === autosaveBaselineSnapshotRef.current) {
      pendingAutosaveSnapshotRef.current = null;
      return;
    }

    pendingAutosaveSnapshotRef.current = null;
    window.setTimeout(() => {
      void handleSaveRef.current();
    }, 0);
  }, []);

  const handleSave = useCallback(async () => {
    if (isInteractionDisabled) {
      return;
    }

    await runSettingsTabAutosave({
      baselineSnapshot: autosaveBaselineSnapshotRef.current,
      currentNodeName,
      flushPendingAutosave,
      nodeConfiguration,
      onSave,
      queuePendingAutosave,
      savingRef,
      selectedIntegration,
      updateAutosaveBaseline,
      validateNow,
    });
  }, [
    isInteractionDisabled,
    validateNow,
    currentNodeName,
    selectedIntegration,
    nodeConfiguration,
    onSave,
    queuePendingAutosave,
    updateAutosaveBaseline,
    flushPendingAutosave,
  ]);

  const handleSaveRef = useRef(handleSave);
  handleSaveRef.current = handleSave;

  const requestAutosave = useCallback(() => {
    if (isInteractionDisabled) {
      return;
    }

    if (autosaveTimerRef.current !== null) {
      window.clearTimeout(autosaveTimerRef.current);
    }
    autosaveTimerRef.current = window.setTimeout(() => {
      autosaveTimerRef.current = null;
      void handleSaveRef.current();
    }, 300);
  }, [isInteractionDisabled]);

  useEffect(() => {
    return () => {
      if (autosaveTimerRef.current !== null) {
        window.clearTimeout(autosaveTimerRef.current);
      }
      if (!isInteractionDisabled) {
        void handleSaveRef.current();
      }
    };
  }, [isInteractionDisabled]);

  useEffect(() => {
    if (
      !shouldRunSettingsTabFallbackAutosave(
        isInteractionDisabled,
        nodeConfiguration,
        currentNodeName,
        selectedIntegration,
        autosaveBaselineSnapshotRef.current,
      )
    ) {
      return;
    }

    const fallbackTimer = window.setTimeout(() => {
      void handleSaveRef.current();
    }, 1200);

    return () => {
      window.clearTimeout(fallbackTimer);
    };
  }, [isInteractionDisabled, nodeConfiguration, currentNodeName, selectedIntegration]);

  return {
    requestAutosave,
    setAutosaveBaseline,
  };
}
