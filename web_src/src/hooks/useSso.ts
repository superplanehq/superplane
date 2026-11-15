import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  ssoActivateConnection,
  ssoCreateDomain,
  ssoCreateScimToken,
  ssoDeleteConnection,
  ssoDeleteDomain,
  ssoDisableConnection,
  ssoGetConnection,
  ssoListDomains,
  ssoListScimTokens,
  ssoPutConnection,
  ssoRevokeScimToken,
  ssoVerifyDomain,
} from "@/api-client/sdk.gen";
import type { SsoPutConnectionRequest } from "@/api-client/types.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export const ssoKeys = {
  all: ["sso"] as const,
  connection: (organizationId: string) => [...ssoKeys.all, "connection", organizationId] as const,
  domains: (organizationId: string) => [...ssoKeys.all, "domains", organizationId] as const,
  tokens: (organizationId: string) => [...ssoKeys.all, "tokens", organizationId] as const,
};

export function useSsoConnection(organizationId: string) {
  return useQuery({
    queryKey: ssoKeys.connection(organizationId),
    enabled: Boolean(organizationId),
    queryFn: async () => {
      const response = await ssoGetConnection(withOrganizationHeader({ organizationId }));
      return response.data;
    },
  });
}

export function useSsoDomains(organizationId: string) {
  return useQuery({
    queryKey: ssoKeys.domains(organizationId),
    enabled: Boolean(organizationId),
    queryFn: async () => {
      const response = await ssoListDomains(withOrganizationHeader({ organizationId }));
      return response.data?.domains ?? [];
    },
  });
}

export function useSsoScimTokens(organizationId: string) {
  return useQuery({
    queryKey: ssoKeys.tokens(organizationId),
    enabled: Boolean(organizationId),
    queryFn: async () => {
      const response = await ssoListScimTokens(withOrganizationHeader({ organizationId }));
      return response.data?.tokens ?? [];
    },
  });
}

function useInvalidateSso(organizationId: string) {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: ssoKeys.connection(organizationId) });
    queryClient.invalidateQueries({ queryKey: ssoKeys.domains(organizationId) });
    queryClient.invalidateQueries({ queryKey: ssoKeys.tokens(organizationId) });
  };
}

export function useSaveSsoConnection(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: (body: SsoPutConnectionRequest) => ssoPutConnection(withOrganizationHeader({ organizationId, body })),
    onSuccess: invalidate,
  });
}

export function useActivateSsoConnection(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: () => ssoActivateConnection(withOrganizationHeader({ organizationId })),
    onSuccess: invalidate,
  });
}

export function useDisableSsoConnection(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: () => ssoDisableConnection(withOrganizationHeader({ organizationId })),
    onSuccess: invalidate,
  });
}

export function useDeleteSsoConnection(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: () => ssoDeleteConnection(withOrganizationHeader({ organizationId })),
    onSuccess: invalidate,
  });
}

export function useCreateSsoDomain(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: (domain: string) => ssoCreateDomain(withOrganizationHeader({ organizationId, body: { domain } })),
    onSuccess: invalidate,
  });
}

export function useVerifySsoDomain(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: (id: string) => ssoVerifyDomain(withOrganizationHeader({ organizationId, path: { id } })),
    onSuccess: invalidate,
  });
}

export function useDeleteSsoDomain(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: (id: string) => ssoDeleteDomain(withOrganizationHeader({ organizationId, path: { id } })),
    onSuccess: invalidate,
  });
}

export function useCreateSsoScimToken(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: (description: string) =>
      ssoCreateScimToken(withOrganizationHeader({ organizationId, body: { description } })),
    onSuccess: invalidate,
  });
}

export function useRevokeSsoScimToken(organizationId: string) {
  const invalidate = useInvalidateSso(organizationId);
  return useMutation({
    mutationFn: (id: string) => ssoRevokeScimToken(withOrganizationHeader({ organizationId, path: { id } })),
    onSuccess: invalidate,
  });
}
