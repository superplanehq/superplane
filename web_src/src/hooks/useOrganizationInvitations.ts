import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  organizationsCancelInvitation,
  organizationsCreateInvitation,
  organizationsListInvitations,
  organizationsResendInvitation,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { organizationKeys } from "./useOrganizationData";

export function useOrganizationInvitations(organizationId: string, enabled: boolean) {
  return useQuery({
    queryKey: organizationKeys.invitations(organizationId),
    queryFn: async () => {
      const response = await organizationsListInvitations(
        withOrganizationHeader({ organizationId, path: { id: organizationId } }),
      );
      if (!response.data?.seats) throw new Error("Could not load invitation capacity. Refresh the page to try again.");
      return response.data;
    },
    enabled: !!organizationId && enabled,
    refetchOnWindowFocus: "always",
  });
}

export function useCreateOrganizationInvitation(organizationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (recipient: { email: string; role: string }) => {
      const response = await organizationsCreateInvitation(
        withOrganizationHeader({ organizationId, path: { id: organizationId }, body: recipient }),
      );
      return response.data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: organizationKeys.invitations(organizationId) }),
  });
}

export function useResendOrganizationInvitation(organizationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (invitationId: string) => {
      const response = await organizationsResendInvitation(
        withOrganizationHeader({ organizationId, path: { id: organizationId, invitationId } }),
      );
      return response.data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: organizationKeys.invitations(organizationId) }),
  });
}

export function useCancelOrganizationInvitation(organizationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (invitationId: string) => {
      await organizationsCancelInvitation(
        withOrganizationHeader({ organizationId, path: { id: organizationId, invitationId } }),
      );
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: organizationKeys.invitations(organizationId) }),
  });
}
