import { useCallback, useEffect, useMemo, useState } from "react";

import {
  clearFleetManagerDraft,
  DEFAULT_FLEET_CAPACITY,
  fleetsMatchConfiguration,
  generateFleetManagerValues,
  loadFleetManagerDraft,
  normalizeInfrastructureFields,
  rememberProviderFields,
  saveFleetManagerDraft,
  validateFleetInfrastructure,
  validateFleetSelection,
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
    fleetCapacities: Object.fromEntries(
      metadata.fleetPresets.map((preset) => [preset.id, { ...DEFAULT_FLEET_CAPACITY }]),
    ),
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
      const defaults = metadata?.infrastructureDefaults.provider === provider ? providerDefaults : {};
      updateDraft((current) => {
        const remembered = rememberProviderFields(current);
        const fields = remembered.fieldsByProvider?.[provider] ?? defaults;
        return {
          ...remembered,
          provider,
          fields,
          fieldsByProvider: { ...remembered.fieldsByProvider, [provider]: fields },
        };
      });
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
        fleetCapacities: current.fleetCapacities[id]
          ? current.fleetCapacities
          : { ...current.fleetCapacities, [id]: { ...DEFAULT_FLEET_CAPACITY } },
      })),
    [updateDraft],
  );

  const setFleetCapacity = useCallback(
    (id: string, capacity: Partial<{ warmCapacity: number; maxCapacity: number }>) =>
      updateDraft((current) => ({
        ...current,
        fleetCapacities: {
          ...current.fleetCapacities,
          [id]: { ...(current.fleetCapacities[id] ?? DEFAULT_FLEET_CAPACITY), ...capacity },
        },
      })),
    [updateDraft],
  );

  const values = useMemo(
    () => (metadata && draft ? generateFleetManagerValues({ metadata, ...draft }) : ""),
    [draft, metadata],
  );
  const fleetErrors = draft ? validateFleetSelection(draft) : [];
  const infrastructureErrors = draft ? validateFleetInfrastructure(draft) : [];
  const validationErrors = [...fleetErrors, ...infrastructureErrors];
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
    fleetErrors,
    infrastructureErrors,
    importedFieldKeys,
    setProvider,
    toggleFleet,
    setFleetCapacity,
    setField,
    setManagerId: (value: string) => updateDraft((current) => ({ ...current, managerId: value })),
    setNamespace: (value: string) => updateDraft((current) => ({ ...current, namespace: value })),
    setServiceAccountName: (value: string) => updateDraft((current) => ({ ...current, serviceAccountName: value })),
    setSuperplaneUrl: (value: string) => updateDraft((current) => ({ ...current, superplaneUrl: value })),
    setRunnerVersion: (value: string) => updateDraft((current) => ({ ...current, runnerVersion: value })),
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
