import type {
  IntegrationCapabilityState,
  IntegrationCapabilityStateState,
  IntegrationProperty,
  OrganizationsIntegration,
} from "@/api-client";
import { useEffect, useMemo, useState } from "react";
import { DEFAULT_CAPABILITY_STATE, groupNodeRefsByCanvas } from "./lib";

function buildCapabilityStates(
  capabilities: IntegrationCapabilityState[] | undefined,
): Record<string, IntegrationCapabilityStateState> {
  const nextStates: Record<string, IntegrationCapabilityStateState> = {};
  (capabilities || []).forEach((capability) => {
    if (!capability.name) return;
    nextStates[capability.name] = capability.state || DEFAULT_CAPABILITY_STATE;
  });
  return nextStates;
}

function buildPropertyDrafts(properties: IntegrationProperty[] | undefined): Record<string, string> {
  const nextDrafts: Record<string, string> = {};
  (properties ?? []).forEach((property, index) => {
    const key = property.name?.trim() || `__property_${index}`;
    nextDrafts[key] = property.value ?? "";
  });
  return nextDrafts;
}

export function useIntegrationDetailsState(integration: OrganizationsIntegration) {
  const capabilities = integration.status?.capabilities;
  const properties = integration.status?.properties;
  const secrets = integration.status?.secrets;
  const usedIn = integration.status?.usedIn;
  const integrationId = integration.metadata?.id;
  const integrationUpdatedAt = integration.metadata?.updatedAt;
  const integrationProperties = useMemo(() => properties ?? [], [properties]);
  const integrationSecrets = useMemo(() => secrets ?? [], [secrets]);
  const workflowGroups = useMemo(() => {
    return groupNodeRefsByCanvas(usedIn ?? []);
  }, [usedIn]);

  const [propertyDrafts, setPropertyDrafts] = useState<Record<string, string>>({});
  const [secretDrafts, setSecretDrafts] = useState<Record<string, string>>({});
  const [capabilityStates, setCapabilityStates] = useState<Record<string, IntegrationCapabilityStateState>>({});

  useEffect(() => {
    setCapabilityStates(buildCapabilityStates(capabilities));
  }, [capabilities]);

  useEffect(() => {
    setPropertyDrafts(buildPropertyDrafts(properties));
  }, [integrationId, integrationUpdatedAt, properties]);

  useEffect(() => {
    setSecretDrafts({});
  }, [integrationId, integrationUpdatedAt, integrationSecrets]);

  return {
    integrationProperties,
    integrationSecrets,
    workflowGroups,
    propertyDrafts,
    setPropertyDrafts,
    secretDrafts,
    setSecretDrafts,
    capabilityStates,
    setCapabilityStates,
  };
}

export type IntegrationDetailsState = ReturnType<typeof useIntegrationDetailsState>;
