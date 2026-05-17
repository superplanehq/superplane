import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  listApps,
  describeApp,
  createApp,
  deleteApp,
  syncApp,
  getAppDashboard,
  updateAppDashboard,
  getAppCanvas,
  listAppDocs,
  getAppDoc,
  updateAppDoc,
  type AppsApp,
  type AppsAppDoc,
  type AppsDashboard,
  type AppsCanvas,
  type DashboardPanel as AppDashboardPanel,
  type DashboardLayoutItem as AppDashboardLayoutItem,
} from "@/lib/appsApi";
import { useOrganizationId } from "./useOrganizationId";

export type { AppsApp, AppsAppDoc, AppsDashboard, AppsCanvas };

// ─── Query Keys ─────────────────────────────────────────────────────────────

export const appKeys = {
  all: ["apps"] as const,
  lists: () => [...appKeys.all, "list"] as const,
  list: (orgId: string) => [...appKeys.lists(), orgId] as const,
  details: () => [...appKeys.all, "detail"] as const,
  detail: (appId: string) => [...appKeys.details(), appId] as const,
  dashboard: (appId: string) => [...appKeys.all, "dashboard", appId] as const,
  canvas: (appId: string) => [...appKeys.all, "canvas", appId] as const,
  docs: (appId: string) => [...appKeys.all, "docs", appId] as const,
  doc: (appId: string, path: string) => [...appKeys.docs(appId), path] as const,
};

// ─── Hooks ──────────────────────────────────────────────────────────────────

export const useApps = (organizationId: string) => {
  return useQuery({
    queryKey: appKeys.list(organizationId),
    queryFn: async () => {
      const result = await listApps(organizationId);
      return result.apps ?? [];
    },
    enabled: !!organizationId,
  });
};

export const useApp = (appId: string) => {
  const organizationId = useOrganizationId() ?? "";
  return useQuery({
    queryKey: appKeys.detail(appId),
    queryFn: async () => {
      const result = await describeApp(appId, organizationId);
      return result.app;
    },
    enabled: !!appId && !!organizationId,
  });
};

export const useCreateApp = (organizationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { displayName: string; appSlug: string; description?: string }) => {
      const result = await createApp(input, organizationId);
      return result.app;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: appKeys.list(organizationId) });
    },
  });
};

export const useDeleteApp = (organizationId: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (appId: string) => {
      await deleteApp(appId, organizationId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: appKeys.list(organizationId) });
    },
  });
};

export const useSyncApp = (appId: string) => {
  const organizationId = useOrganizationId() ?? "";
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const result = await syncApp(appId, organizationId);
      return result.app;
    },
    onSuccess: (data) => {
      if (data) {
        queryClient.setQueryData(appKeys.detail(appId), data);
      }
    },
  });
};

export const useAppDashboard = (appId: string, enabled: boolean = true) => {
  const organizationId = useOrganizationId() ?? "";
  return useQuery({
    queryKey: appKeys.dashboard(appId),
    queryFn: async () => {
      const result = await getAppDashboard(appId, organizationId);
      return result.dashboard;
    },
    enabled: enabled && !!appId && !!organizationId,
    staleTime: 30_000,
  });
};

export const useUpdateAppDashboard = (appId: string) => {
  const organizationId = useOrganizationId() ?? "";
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { panels: AppDashboardPanel[]; layout: AppDashboardLayoutItem[] }) => {
      const result = await updateAppDashboard(appId, input, organizationId);
      return result.dashboard;
    },
    onSuccess: (data) => {
      queryClient.setQueryData(appKeys.dashboard(appId), data);
    },
  });
};

export const useAppCanvas = (appId: string, enabled: boolean = true) => {
  const organizationId = useOrganizationId() ?? "";
  return useQuery({
    queryKey: appKeys.canvas(appId),
    queryFn: async () => {
      const result = await getAppCanvas(appId, organizationId);
      return result.canvas;
    },
    enabled: enabled && !!appId && !!organizationId,
  });
};

export const useAppDocs = (appId: string, enabled: boolean = true) => {
  const organizationId = useOrganizationId() ?? "";
  return useQuery({
    queryKey: appKeys.docs(appId),
    queryFn: async () => {
      const result = await listAppDocs(appId, organizationId);
      return result.docs ?? [];
    },
    enabled: enabled && !!appId && !!organizationId,
  });
};

export const useAppDoc = (appId: string, path: string, enabled: boolean = true) => {
  const organizationId = useOrganizationId() ?? "";
  return useQuery({
    queryKey: appKeys.doc(appId, path),
    queryFn: async () => {
      const result = await getAppDoc(appId, path, organizationId);
      return result.doc;
    },
    enabled: enabled && !!appId && !!path && !!organizationId,
  });
};

export const useUpdateAppDoc = (appId: string) => {
  const organizationId = useOrganizationId() ?? "";
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { path: string; content: string }) => {
      const result = await updateAppDoc(appId, input.path, input.content, organizationId);
      return result.doc;
    },
    onSuccess: (data) => {
      if (data?.path) {
        queryClient.setQueryData(appKeys.doc(appId, data.path), data);
        queryClient.invalidateQueries({ queryKey: appKeys.docs(appId) });
      }
    },
  });
};

// ─── Re-exported types for local use ─────────────────────────────────────────

export type AppDashboardQueryResult = ReturnType<typeof useAppDashboard>;
export type UpdateAppDashboardMutationResult = ReturnType<typeof useUpdateAppDashboard>;
