import { useCallback, useEffect, useMemo, useState } from "react";

import {
  clearFleetManagerDraft,
  fleetsMatchConfiguration,
  generateFleetManagerValues,
  loadFleetManagerDraft,
  normalizeInfrastructureFields,
  saveFleetManagerDraft,
  validateFleetManagerConfiguration,
  type FleetManagerDraft,
  type FleetManagerProvider,
  type FleetManagerSetupMetadata,
  type InstalledFleet,
} from "@/lib/fleetManagerSetup";

const VERIFICATION_INTERVAL_MS = 5_000;
const VERIFICATION_TIMEOUT_MS = 10 * 60_000;

function defaultDraft(metadata: FleetManagerSetupMetadata): FleetManagerDraft {
  const defaults = metadata.infrastructureDefaults;
  const { superplaneUrl, ...fields } = normalizeInfrastructureFields(defaults.fields);
  return {
    provider: defaults.provider ?? "aws",
    managerId: defaults.managerId || "my-cluster",
    namespace: defaults.namespace || "superplane",
    serviceAccountName: defaults.serviceAccountName || "superplane-fleet-manager",
    superplaneUrl: superplaneUrl || window.location.origin,
    runnerVersion: metadata.defaultRunnerVersion,
    selectedFleetIds: metadata.fleetPresets[0] ? [metadata.fleetPresets[0].id] : [],
    warmCapacity: 0,
    maxCapacity: 1,
    fields,
  };
}

export function useFleetManagerSetup(enabled = true) {
  const [metadata, setMetadata] = useState<FleetManagerSetupMetadata | null>(null);
  const [draft, setDraft] = useState<FleetManagerDraft | null>(null);
  const [loadError, setLoadError] = useState(false);

  useEffect(() => {
    if (!enabled || metadata) return;

    const controller = new AbortController();
    setLoadError(false);
    fetch("/admin/api/installation/fleet-manager/setup", {
      credentials: "include",
      signal: controller.signal,
    })
      .then(async (response) => {
        if (!response.ok) throw new Error("Could not load Fleet Manager setup metadata");
        return (await response.json()) as FleetManagerSetupMetadata;
      })
      .then((body) => {
        const initialDraft = loadFleetManagerDraft() ?? defaultDraft(body);
        setMetadata(body);
        saveFleetManagerDraft(initialDraft);
        setDraft(initialDraft);
      })
      .catch((error: unknown) => {
        if (!(error instanceof DOMException && error.name === "AbortError")) setLoadError(true);
      });

    return () => controller.abort();
  }, [enabled, metadata]);

  const updateDraft = useCallback((update: (current: FleetManagerDraft) => FleetManagerDraft) => {
    setDraft((current) => {
      if (!current) return current;
      const next = update(current);
      saveFleetManagerDraft(next);
      return next;
    });
  }, []);

  const setProvider = useCallback(
    (provider: FleetManagerProvider) => {
      const providerDefaults = normalizeInfrastructureFields(metadata?.infrastructureDefaults.fields ?? {});
      delete providerDefaults.superplaneUrl;
      updateDraft((current) => ({
        ...current,
        provider,
        fields: metadata?.infrastructureDefaults.provider === provider ? providerDefaults : {},
      }));
    },
    [metadata, updateDraft],
  );

  const setField = useCallback(
    (key: string, value: string) =>
      updateDraft((current) => ({ ...current, fields: { ...current.fields, [key]: value } })),
    [updateDraft],
  );

  const toggleFleet = useCallback(
    (id: string, checked: boolean) =>
      updateDraft((current) => ({
        ...current,
        selectedFleetIds: checked
          ? [...new Set([...current.selectedFleetIds, id])]
          : current.selectedFleetIds.filter((item) => item !== id),
      })),
    [updateDraft],
  );

  const values = useMemo(
    () => (metadata && draft ? generateFleetManagerValues({ metadata, ...draft }) : ""),
    [draft, metadata],
  );
  const validationErrors = draft ? validateFleetManagerConfiguration(draft) : [];
  const importedFieldKeys = useMemo(() => {
    if (!metadata || !draft || metadata.infrastructureDefaults.provider !== draft.provider) return new Set<string>();
    const defaults = normalizeInfrastructureFields(metadata.infrastructureDefaults.fields);
    return new Set(Object.keys(defaults).filter((key) => defaults[key] === draft.fields[key]));
  }, [draft, metadata]);

  return {
    metadata,
    draft,
    loadError,
    values,
    validationErrors,
    importedFieldKeys,
    setProvider,
    toggleFleet,
    setField,
    setManagerId: (value: string) => updateDraft((current) => ({ ...current, managerId: value })),
    setNamespace: (value: string) => updateDraft((current) => ({ ...current, namespace: value })),
    setServiceAccountName: (value: string) => updateDraft((current) => ({ ...current, serviceAccountName: value })),
    setSuperplaneUrl: (value: string) => updateDraft((current) => ({ ...current, superplaneUrl: value })),
    setRunnerVersion: (value: string) => updateDraft((current) => ({ ...current, runnerVersion: value })),
    setWarmCapacity: (value: number) => updateDraft((current) => ({ ...current, warmCapacity: value })),
    setMaxCapacity: (value: number) => updateDraft((current) => ({ ...current, maxCapacity: value })),
  };
}

export type FleetManagerVerificationStatus =
  | "idle"
  | "checking"
  | "waiting"
  | "verified"
  | "unauthorized"
  | "error"
  | "timeout";

export function useFleetManagerVerification(
  enabled: boolean,
  draft: FleetManagerDraft | null,
  metadata: FleetManagerSetupMetadata | null,
) {
  const [status, setStatus] = useState<FleetManagerVerificationStatus>("idle");
  const [retryKey, setRetryKey] = useState(0);

  useEffect(() => {
    if (!enabled || !draft || !metadata) {
      setStatus("idle");
      return;
    }

    const controller = new AbortController();
    const startedAt = Date.now();
    let timeout: number | undefined;

    const check = async () => {
      setStatus((current) => (current === "waiting" ? current : "checking"));
      try {
        const response = await fetch("/admin/api/installation/fleets", {
          credentials: "include",
          signal: controller.signal,
        });
        if (response.status === 401 || response.status === 403) {
          setStatus("unauthorized");
          return;
        }
        if (!response.ok) {
          setStatus("error");
          return;
        }

        const body = (await response.json()) as { fleets?: InstalledFleet[] };
        if (fleetsMatchConfiguration(body.fleets ?? [], draft, metadata.fleetPresets)) {
          clearFleetManagerDraft();
          setStatus("verified");
          return;
        }
        if (Date.now() - startedAt >= VERIFICATION_TIMEOUT_MS) {
          setStatus("timeout");
          return;
        }

        setStatus("waiting");
        timeout = window.setTimeout(() => void check(), VERIFICATION_INTERVAL_MS);
      } catch (error: unknown) {
        if (!(error instanceof DOMException && error.name === "AbortError")) setStatus("error");
      }
    };

    void check();
    return () => {
      controller.abort();
      if (timeout !== undefined) window.clearTimeout(timeout);
    };
  }, [draft, enabled, metadata, retryKey]);

  return { status, retry: () => setRetryKey((current) => current + 1) };
}
