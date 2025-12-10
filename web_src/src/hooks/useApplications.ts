import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  applicationsListApplications,
  organizationsListApplications,
  organizationsInstallApplication,
  organizationsUpdateApplication,
  organizationsUninstallApplication,
} from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import type { OrganizationsInstallApplicationBody } from "@/api-client/types.gen";

export const applicationKeys = {
  all: ["applications"] as const,
  available: () => [...applicationKeys.all, "available"] as const,
  installed: (organizationId: string) => [...applicationKeys.all, "installed", organizationId] as const,
  installation: (organizationId: string, installationId: string) =>
    [...applicationKeys.installed(organizationId), installationId] as const,
};

// Hook to fetch available applications (catalog)
export const useAvailableApplications = () => {
  return useQuery({
    queryKey: applicationKeys.available(),
    queryFn: async () => {
      const response = await applicationsListApplications(withOrganizationHeader({}));
      return response.data?.applications || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
  });
};

// Hook to fetch installed applications for an organization
export const useInstalledApplications = (organizationId: string) => {
  return useQuery({
    queryKey: applicationKeys.installed(organizationId),
    queryFn: async () => {
      const response = await organizationsListApplications(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      return response.data?.applications || [];
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!organizationId,
  });
};

// Hook to fetch a single application installation
export const useApplicationInstallation = (organizationId: string, installationId: string) => {
  return useQuery({
    queryKey: applicationKeys.installation(organizationId, installationId),
    queryFn: async () => {
      const response = await organizationsListApplications(
        withOrganizationHeader({
          path: { id: organizationId },
        }),
      );
      const installations = response.data?.applications || [];
      return installations.find((app) => app.id === installationId) || null;
    },
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!organizationId && !!installationId,
  });
};

// Hook to install an application
export const useInstallApplication = (organizationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: {
      appName: string;
      installationName: string;
      configuration?: Record<string, unknown>;
    }) => {
      return await organizationsInstallApplication(
        withOrganizationHeader({
          path: { id: organizationId },
          body: {
            appName: data.appName,
            installationName: data.installationName,
            configuration: data.configuration,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: applicationKeys.installed(organizationId),
      });
    },
  });
};

// Hook to update an application installation
export const useUpdateApplication = (organizationId: string, installationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (configuration: Record<string, unknown>) => {
      return await organizationsUpdateApplication(
        withOrganizationHeader({
          path: { id: organizationId, installationId: installationId },
          body: {
            configuration,
          },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: applicationKeys.installed(organizationId),
      });
      queryClient.invalidateQueries({
        queryKey: applicationKeys.installation(organizationId, installationId),
      });
    },
  });
};

// Hook to uninstall an application
export const useUninstallApplication = (organizationId: string, installationId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      return await organizationsUninstallApplication(
        withOrganizationHeader({
          path: { id: organizationId, installationId: installationId },
        }),
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: applicationKeys.installed(organizationId),
      });
      queryClient.removeQueries({
        queryKey: applicationKeys.installation(organizationId, installationId),
      });
    },
  });
};
