import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocation } from "react-router";
import {
  organizationsListManagedRunnerFleets,
  organizationsDescribeRunnerFleet,
  organizationsGetRunnerFleetCapacity,
  organizationsListFleetRunners,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { FEATURE_ORGANIZATION_RUNNERS } from "@/lib/experimentalFeatures";
import { useExperimentalFeature } from "./useExperimentalFeature";
import { experimentalFeaturesKeys } from "./useExperimentalFeatures";
import { organizationKeys, useOrganization } from "./useOrganizationData";

export const organizationFleetKeys = {
  all: (organizationId: string) => ["organization-fleet-management", organizationId] as const,
};
const activeQuery = { staleTime: 0, gcTime: 0, refetchInterval: 5000, retry: false } as const;

export function useManagedOrganizationFleets(organizationId: string, active = true) {
  const { hasPermission, isLoading } = usePermissions();
  return useQuery({
    ...activeQuery,
    refetchInterval: active ? 5000 : false,
    queryKey: [...organizationFleetKeys.all(organizationId), "catalog"],
    enabled: !!organizationId && !isLoading && hasPermission("org", "read"),
    queryFn: async ({ signal }) => {
      const response = await organizationsListManagedRunnerFleets(
        withOrganizationHeader({ organizationId, path: { id: organizationId }, signal }),
      );
      return response.data.fleets ?? [];
    },
  });
}

export function useOrganizationFleetAccess(organizationId: string) {
  const { pathname } = useLocation();
  const active = pathname.split("/").includes("fleets");
  const { hasPermission, canAct, isLoading } = usePermissions();
  const readable = !isLoading && hasPermission("org", "read");
  const organization = useOrganization(organizationId, readable, active ? 5000 : 0);
  const feature = useExperimentalFeature(organizationId);
  const catalog = useManagedOrganizationFleets(organizationId, active);
  const resolved = organization.isSuccess && feature.organizationReady && !feature.isLoading && !feature.lookupFailed;
  const enabled = resolved && feature.has(FEATURE_ORGANIZATION_RUNNERS);
  return {
    catalog,
    enabled,
    canConfigure: canAct("org", "update") && enabled && catalog.isSuccess,
    canTerminate: canAct("org", "update"),
    showNavigation: readable && !(resolved && !enabled && catalog.isSuccess && catalog.data.length === 0),
    accessError: organization.isError || feature.lookupFailed,
    resolved,
  };
}

export function useOrganizationFleet(organizationId: string, fleetId: string) {
  return useQuery({
    ...activeQuery,
    queryKey: [...organizationFleetKeys.all(organizationId), fleetId, "detail"],
    queryFn: async ({ signal }) => {
      const response = await organizationsDescribeRunnerFleet(
        withOrganizationHeader({ organizationId, path: { id: organizationId, fleetId }, signal }),
      );
      if (!response.data.fleet) throw new Error("Fleet unavailable");
      return response.data.fleet;
    },
  });
}

export function useOrganizationFleetCapacity(organizationId: string, fleetId: string) {
  return useQuery({
    ...activeQuery,
    queryKey: [...organizationFleetKeys.all(organizationId), fleetId, "capacity"],
    queryFn: async ({ signal }) => {
      const response = await organizationsGetRunnerFleetCapacity(
        withOrganizationHeader({ organizationId, path: { id: organizationId, fleetId }, signal }),
      );
      return response.data;
    },
  });
}

export function useOrganizationFleetRunners(organizationId: string, fleetId: string, state: string, afterId?: string) {
  return useQuery({
    ...activeQuery,
    queryKey: [...organizationFleetKeys.all(organizationId), fleetId, "runners", state, afterId],
    queryFn: async ({ signal }) => {
      const response = await organizationsListFleetRunners(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          query: { fleetId, states: [state], limit: 50, afterId },
          signal,
        }),
      );
      return response.data;
    },
  });
}

export function useRefreshOrganizationFleets(organizationId: string) {
  const client = useQueryClient();
  return () => {
    void client.invalidateQueries({ queryKey: organizationFleetKeys.all(organizationId) });
    void client.invalidateQueries({ queryKey: organizationKeys.details(organizationId) });
    void client.invalidateQueries({ queryKey: experimentalFeaturesKeys.registry() });
  };
}
