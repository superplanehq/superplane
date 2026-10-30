import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useCreateUserToken } from "@/hooks/useUserTokens";

import {
  clearFleetManagerDraft,
  DEFAULT_FLEET_CAPACITY,
  DEFAULT_SETUP_FLEET_ID,
  fleetImage,
  fleetsMatchConfiguration,
  generateFleetManagerValues,
  installationRunnerVersion,
  loadFleetManagerDraft,
  mergeTerraformInfrastructure,
  normalizeInfrastructureFields,
  rememberProviderFields,
  resolveSetupMachineType,
  saveFleetManagerDraft,
  terraformInstallPlaceholders,
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
  const provider = defaults.provider ?? "aws";
  const { superplaneUrl, ...infrastructureFields } = normalizeInfrastructureFields(defaults.fields);
  const fields = withRecommendedMachineTypes(infrastructureFields, provider);
  const hasDefaultFleet = metadata.fleetPresets.some((preset) => preset.id === DEFAULT_SETUP_FLEET_ID);
  return {
    provider,
    managerId: defaults.managerId || "my-cluster",
    namespace: defaults.namespace || "superplane",
    serviceAccountName: defaults.serviceAccountName || "superplane-fleet-manager",
    superplaneUrl: superplaneUrl || window.location.origin,
    runnerVersion: installationRunnerVersion(metadata),
    selectedFleetIds: hasDefaultFleet ? [DEFAULT_SETUP_FLEET_ID] : [],
    fleetCapacities: hasDefaultFleet ? { [DEFAULT_SETUP_FLEET_ID]: { ...DEFAULT_FLEET_CAPACITY } } : {},
    fields,
    kubeContext: "",
  };
}

function withRecommendedMachineTypes(
  fields: Record<string, string>,
  provider: FleetManagerProvider,
): Record<string, string> {
  if (provider === "docker") return fields;
  const key = `machineType.${DEFAULT_SETUP_FLEET_ID}`;
  return { ...fields, [key]: resolveSetupMachineType(provider, fields[key]) };
}

function pinSetupFleet(draft: FleetManagerDraft, metadata: FleetManagerSetupMetadata): FleetManagerDraft {
  const hasDefaultFleet = metadata.fleetPresets.some((preset) => preset.id === DEFAULT_SETUP_FLEET_ID);
  if (!hasDefaultFleet) return { ...draft, selectedFleetIds: [] };
  return {
    ...draft,
    selectedFleetIds: [DEFAULT_SETUP_FLEET_ID],
    fleetCapacities: {
      ...draft.fleetCapacities,
      [DEFAULT_SETUP_FLEET_ID]: draft.fleetCapacities[DEFAULT_SETUP_FLEET_ID] ?? { ...DEFAULT_FLEET_CAPACITY },
    },
    fields: withRecommendedMachineTypes(draft.fields, draft.provider),
  };
}

function draftFromMetadata(metadata: FleetManagerSetupMetadata): FleetManagerDraft {
  metadata.runnerVersions = metadata.runnerVersions?.length
    ? metadata.runnerVersions
    : metadata.defaultRunnerVersion
      ? [metadata.defaultRunnerVersion]
      : [];
  const restoredDraft = loadFleetManagerDraft();
  let draft = restoredDraft ? mergeTerraformInfrastructure(restoredDraft, metadata) : defaultDraft(metadata);
  if (draft.provider === "docker") {
    draft = draftForProvider(draft, "aws", metadata);
  }
  return pinSetupFleet({ ...draft, runnerVersion: installationRunnerVersion(metadata) }, metadata);
}

function draftForProvider(
  current: FleetManagerDraft,
  provider: FleetManagerProvider,
  metadata: FleetManagerSetupMetadata | null,
): FleetManagerDraft {
  const providerDefaults = normalizeInfrastructureFields(metadata?.infrastructureDefaults.fields ?? {});
  delete providerDefaults.superplaneUrl;
  const defaults = metadata?.infrastructureDefaults.provider === provider ? providerDefaults : {};
  const remembered = rememberProviderFields(current);
  const base = remembered.fieldsByProvider?.[provider] ?? defaults;
  const fields = withRecommendedMachineTypes(base, provider);
  return {
    ...remembered,
    provider,
    fields,
    fieldsByProvider: { ...remembered.fieldsByProvider, [provider]: fields },
  };
}

function missingImageErrors(draft: FleetManagerDraft, metadata: FleetManagerSetupMetadata): string[] {
  if (draft.provider === "docker" || metadata.infrastructureDefaults.source !== "terraform") return [];
  return draft.selectedFleetIds.flatMap((fleetId) => {
    const preset = metadata.fleetPresets.find((candidate) => candidate.id === fleetId);
    if (!preset || fleetImage(draft.fields, preset.spec.architecture)) return [];
    return [`Add the ${preset.spec.architecture} runner image through Terraform before you select ${fleetId}.`];
  });
}

function fleetSelectionErrors(draft: FleetManagerDraft | null, metadata: FleetManagerSetupMetadata | null): string[] {
  if (!draft) return [];
  const errors = validateFleetSelection({ ...draft, runnerVersions: metadata?.runnerVersions ?? [] });
  if (!metadata) return errors;
  return [...errors, ...missingImageErrors(draft, metadata), ...terraformInstallPlaceholders(draft, metadata)];
}

function useFleetManagerDraft(enabled: boolean) {
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
        const initialDraft = draftFromMetadata(body);
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

  return { metadata, draft, loadError, updateDraft };
}

function useFleetManagerToken(organizationId: string) {
  const [tokenState, setTokenState] = useState<{ organizationId: string; plaintext: string } | null>(null);
  const [tokenError, setTokenError] = useState<string | null>(null);
  const createToken = useCreateUserToken(organizationId);
  const plaintext = tokenState?.organizationId === organizationId ? tokenState.plaintext : "";
  const inFlight = useRef<Promise<boolean> | null>(null);

  const prepareInstallation = useCallback(async (): Promise<boolean> => {
    if (plaintext) return true;
    if (inFlight.current) return inFlight.current;
    if (!organizationId) {
      setTokenError("SuperPlane could not identify the organization for this personal token.");
      return false;
    }

    setTokenError(null);
    const run = (async () => {
      try {
        const response = await createToken.mutateAsync({ name: "Fleet Manager" });
        const created = response?.plaintext?.trim();
        if (!created) throw new Error("Personal token response did not include plaintext");
        setTokenState({ organizationId, plaintext: created });
        return true;
      } catch {
        setTokenError("SuperPlane could not create the Fleet Manager personal token. Try again.");
        return false;
      } finally {
        inFlight.current = null;
      }
    })();
    inFlight.current = run;
    return run;
  }, [createToken, organizationId, plaintext]);

  return {
    plaintext,
    tokenError,
    isPreparingInstallation: createToken.isPending,
    prepareInstallation,
  };
}

export function useFleetManagerSetup(enabled = true, organizationId = "") {
  const { metadata, draft, loadError, updateDraft } = useFleetManagerDraft(enabled);
  const token = useFleetManagerToken(organizationId);

  const setProvider = useCallback(
    (provider: FleetManagerProvider) => updateDraft((current) => draftForProvider(current, provider, metadata)),
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
    () =>
      metadata && draft && token.plaintext && draft.provider !== "docker"
        ? generateFleetManagerValues({ metadata, ...draft, installationAdminToken: token.plaintext })
        : "",
    [draft, metadata, token.plaintext],
  );
  const fleetErrors = fleetSelectionErrors(draft, metadata);
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
    installationToken: token.plaintext,
    tokenError: token.tokenError,
    isPreparingInstallation: token.isPreparingInstallation,
    validationErrors,
    fleetErrors,
    infrastructureErrors,
    importedFieldKeys,
    prepareInstallation: token.prepareInstallation,
    hasFleetImage: (architecture: "amd64" | "arm64") => Boolean(draft && fleetImage(draft.fields, architecture)),
    setProvider,
    toggleFleet,
    setFleetCapacity,
    setField,
    setManagerId: (value: string) => updateDraft((current) => ({ ...current, managerId: value })),
    setNamespace: (value: string) => updateDraft((current) => ({ ...current, namespace: value })),
    setServiceAccountName: (value: string) => updateDraft((current) => ({ ...current, serviceAccountName: value })),
    setSuperplaneUrl: (value: string) => updateDraft((current) => ({ ...current, superplaneUrl: value })),
    setKubeContext: (value: string) =>
      updateDraft((current) => ({ ...current, kubeContext: value.replace(/[\r\n]/g, "") })),
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

function installationKey(draft: FleetManagerDraft): string {
  return JSON.stringify({ ...draft, kubeContext: undefined });
}

// The Kubernetes context selects the Helm cluster. The fleet check does not read it.
export function sameFleetInstallation(left: FleetManagerDraft | null, right: FleetManagerDraft | null): boolean {
  if (left === right) return true;
  if (!left || !right) return false;
  return installationKey(left) === installationKey(right);
}

function useFleetInstallation(draft: FleetManagerDraft | null): FleetManagerDraft | null {
  const [installation, setInstallation] = useState(draft);
  if (!sameFleetInstallation(installation, draft)) setInstallation(draft);
  return installation;
}

export function useFleetManagerVerification(
  enabled: boolean,
  draft: FleetManagerDraft | null,
  metadata: FleetManagerSetupMetadata | null,
) {
  const [status, setStatus] = useState<FleetManagerVerificationStatus>("idle");
  const [retryKey, setRetryKey] = useState(0);
  const installation = useFleetInstallation(draft);

  useEffect(() => {
    if (!enabled || !installation || !metadata) {
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
        const installed = body.fleets ?? [];
        const matches = fleetsMatchConfiguration(installed, installation, metadata.fleetPresets);
        if (matches) {
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
  }, [enabled, installation, metadata, retryKey]);

  return { status, retry: () => setRetryKey((current) => current + 1) };
}
