import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  organizationsGetOktaSettings,
  organizationsUpdateOktaSettings,
  organizationsRotateOktaScimToken,
} from "../api-client/sdk.gen";
import type {
  OrganizationsGetOktaSettingsResponse,
  OrganizationsUpdateOktaSettingsBody,
  OrganizationsRotateOktaScimTokenBody,
} from "../api-client/types.gen";

export function useOktaSettings(organizationId: string) {
  const queryClient = useQueryClient();

  const query = useQuery<OrganizationsGetOktaSettingsResponse>({
    queryKey: ["okta-settings", organizationId],
    queryFn: async () => {
      const res = await organizationsGetOktaSettings({
        path: { id: organizationId },
        headers: {
          "x-organization-id": organizationId,
        },
      });
      return res.data;
    },
  });

  const updateMutation = useMutation({
    mutationFn: async (body: OrganizationsUpdateOktaSettingsBody) => {
      return organizationsUpdateOktaSettings({
        path: { id: organizationId },
        headers: {
          "x-organization-id": organizationId,
        },
        body,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["okta-settings", organizationId] });
    },
  });

  const rotateTokenMutation = useMutation({
    mutationFn: async (body: OrganizationsRotateOktaScimTokenBody) => {
      const res = await organizationsRotateOktaScimToken({
        path: { id: organizationId },
        headers: {
          "x-organization-id": organizationId,
        },
        body,
      });
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["okta-settings", organizationId] });
    },
  });

  return {
    oktaSettings: query.data?.settings,
    isLoading: query.isLoading,
    error: query.error,
    updateSettings: updateMutation.mutateAsync,
    isUpdating: updateMutation.isPending,
    rotateToken: rotateTokenMutation.mutateAsync,
    isRotating: rotateTokenMutation.isPending,
    rotatedToken: rotateTokenMutation.data?.token,
  };
}
