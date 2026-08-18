import {
  meDescribeNotificationSettings,
  meUpdateNotificationSettings,
  type MeNotificationSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const notificationSettingsKey = (organizationId: string) => ["me", organizationId, "notification-settings"] as const;

export function useNotificationSettings(organizationId: string) {
  return useQuery({
    queryKey: notificationSettingsKey(organizationId),
    queryFn: async (): Promise<MeNotificationSettings> => {
      const response = await meDescribeNotificationSettings(withOrganizationHeader({ organizationId }));
      return response.data?.settings ?? {};
    },
    enabled: Boolean(organizationId),
  });
}

export function useUpdateNotificationSettings(organizationId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (settings: MeNotificationSettings) => {
      const response = await meUpdateNotificationSettings(
        withOrganizationHeader({
          organizationId,
          body: { settings },
        }),
      );
      if (!response.data?.settings) {
        throw new Error("Failed to update notification settings");
      }
      return response.data.settings;
    },
    onSuccess: (settings) => {
      queryClient.setQueryData(notificationSettingsKey(organizationId), settings);
    },
  });
}
