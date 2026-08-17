import {
  factoriesDescribeNotificationSettings,
  factoriesUpdateNotificationSettings,
  type FactoriesNotificationSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const notificationSettingsKey = (organizationId: string) =>
  ["factories", organizationId, "notification-settings"] as const;

export function useNotificationSettings(organizationId: string) {
  return useQuery({
    queryKey: notificationSettingsKey(organizationId),
    queryFn: async (): Promise<FactoriesNotificationSettings> => {
      const response = await factoriesDescribeNotificationSettings(withOrganizationHeader({ organizationId }));
      return response.data?.settings ?? {};
    },
    enabled: Boolean(organizationId),
  });
}

export function useUpdateNotificationSettings(organizationId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (settings: FactoriesNotificationSettings) => {
      const response = await factoriesUpdateNotificationSettings(
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
