import { useEffect, useMemo, useState } from "react";

import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import { useCreateUserToken } from "@/hooks/useUserTokens";
import {
  generateFleetManagerValues,
  validateFleetManagerConfiguration,
  type FleetManagerProvider,
  type FleetManagerSetupMetadata,
} from "@/lib/fleetManagerSetup";

function useFleetManagerMetadata() {
  const [metadata, setMetadata] = useState<FleetManagerSetupMetadata | null>(null);
  const [loadError, setLoadError] = useState(false);

  useEffect(() => {
    let active = true;
    fetch("/admin/api/installation/fleet-manager/setup", { credentials: "include" })
      .then(async (response) => {
        if (!response.ok) throw new Error("Could not load Fleet Manager setup metadata");
        return (await response.json()) as FleetManagerSetupMetadata;
      })
      .then((body) => active && setMetadata(body))
      .catch(() => active && setLoadError(true));
    return () => {
      active = false;
    };
  }, []);

  return { metadata, loadError };
}

export function useFleetManagerSetup() {
  const { metadata, loadError } = useFleetManagerMetadata();
  const [provider, setProvider] = useState<FleetManagerProvider>("aws");
  const [managerId, setManagerId] = useState("my-cluster");
  const [namespace, setNamespace] = useState("superplane");
  const [serviceAccountName, setServiceAccountName] = useState("superplane-fleet-manager");
  const [runnerVersion, setRunnerVersion] = useState("");
  const [selectedFleetIds, setSelectedFleetIds] = useState<string[]>([]);
  const [warmCapacity, setWarmCapacity] = useState(0);
  const [maxCapacity, setMaxCapacity] = useState(1);
  const [providerFieldValues, setProviderFieldValues] = useState<Record<FleetManagerProvider, Record<string, string>>>({
    aws: {},
    gcp: {},
    azure: {},
  });
  const [plaintextToken, setPlaintextToken] = useState<string | null>(null);
  const [tokenError, setTokenError] = useState<string | null>(null);
  const organizationId = useAccountOrganizations().data?.[0]?.id || "";
  const createToken = useCreateUserToken(organizationId);
  const fields = providerFieldValues[provider];

  useEffect(() => {
    if (!metadata) return;
    setRunnerVersion(metadata.defaultRunnerVersion);
    setSelectedFleetIds(metadata.fleetPresets[0] ? [metadata.fleetPresets[0].id] : []);
  }, [metadata]);

  const values = useMemo(
    () =>
      metadata
        ? generateFleetManagerValues({
            metadata,
            provider,
            managerId,
            runnerVersion,
            selectedFleetIds,
            warmCapacity,
            maxCapacity,
            fields,
            serviceAccountName,
          })
        : "",
    [
      fields,
      managerId,
      maxCapacity,
      metadata,
      provider,
      runnerVersion,
      selectedFleetIds,
      serviceAccountName,
      warmCapacity,
    ],
  );
  const validationErrors = validateFleetManagerConfiguration({
    managerId,
    namespace,
    serviceAccountName,
    runnerVersion,
    selectedFleetIds,
    warmCapacity,
    maxCapacity,
  });

  const toggleFleet = (id: string, checked: boolean) =>
    setSelectedFleetIds((current) =>
      checked ? [...new Set([...current, id])] : current.filter((item) => item !== id),
    );
  const setField = (key: string, value: string) =>
    setProviderFieldValues((current) => ({
      ...current,
      [provider]: { ...current[provider], [key]: value },
    }));
  const generateToken = async () => {
    setTokenError(null);
    try {
      const token = await createToken.mutateAsync({ name: "Fleet Manager" });
      if (!token?.plaintext) throw new Error("token response has no plaintext value");
      setPlaintextToken(token.plaintext);
    } catch {
      setTokenError("SuperPlane could not create the token. Try again.");
    }
  };

  return {
    metadata,
    loadError,
    provider,
    setProvider,
    managerId,
    setManagerId,
    namespace,
    setNamespace,
    serviceAccountName,
    setServiceAccountName,
    runnerVersion,
    setRunnerVersion,
    selectedFleetIds,
    toggleFleet,
    warmCapacity,
    setWarmCapacity,
    maxCapacity,
    setMaxCapacity,
    fields,
    setField,
    values,
    validationErrors,
    organizationId,
    tokenPending: createToken.isPending,
    tokenError,
    plaintextToken,
    generateToken,
  };
}
