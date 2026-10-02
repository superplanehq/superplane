import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { AdminIntakeEntry } from "@/pages/admin/intakes/intakeCatalogModel";

export interface AdminIntakePreview {
  key: string;
  status: string;
  organization_added: boolean;
  available: boolean;
}

export interface CreateAdminIntakeInput {
  key: string;
  name: string;
  category: string;
  status_note: string;
}

export interface UpdateAdminIntakeInput {
  name?: string;
  category?: string;
  status?: string;
  status_note?: string;
  enabled_for_all?: boolean;
}

export const adminIntakeCatalogKeys = {
  all: ["adminIntakeCatalog"] as const,
  list: () => [...adminIntakeCatalogKeys.all, "list"] as const,
  preview: (key: string, organizationId: string) =>
    [...adminIntakeCatalogKeys.all, "preview", key, organizationId] as const,
};

const BASE_PATH = "/admin/api/intake-catalog";

async function requestJSON<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    credentials: "include",
    ...init,
    headers: init.body ? { "Content-Type": "application/json", ...init.headers } : init.headers,
  });
  if (!response.ok) {
    const message = (await response.text()).trim();
    throw new Error(message || `Request failed (${response.status})`);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

function entryPath(key: string): string {
  return `${BASE_PATH}/${encodeURIComponent(key)}`;
}

async function fetchAdminIntakeCatalog(): Promise<AdminIntakeEntry[]> {
  const data = await requestJSON<{ entries?: AdminIntakeEntry[] }>(BASE_PATH);
  return data.entries ?? [];
}

export function useAdminIntakeCatalog() {
  return useQuery({
    queryKey: adminIntakeCatalogKeys.list(),
    queryFn: fetchAdminIntakeCatalog,
    staleTime: 30 * 1000,
  });
}

/** Pass an empty organization id to preview a company without access. */
export function useAdminIntakePreview(key: string, organizationId: string) {
  return useQuery({
    queryKey: adminIntakeCatalogKeys.preview(key, organizationId),
    queryFn: () => {
      const params = organizationId ? `?${new URLSearchParams({ organization_id: organizationId })}` : "";
      return requestJSON<AdminIntakePreview>(`${entryPath(key)}/preview${params}`);
    },
    enabled: Boolean(key),
  });
}

function replaceEntry(entries: AdminIntakeEntry[] | undefined, entry: AdminIntakeEntry): AdminIntakeEntry[] {
  const list = entries ?? [];
  if (!list.some((item) => item.key === entry.key)) {
    return [...list, entry];
  }
  return list.map((item) => (item.key === entry.key ? entry : item));
}

function useEntryMutation<TVariables>(mutationFn: (variables: TVariables) => Promise<AdminIntakeEntry>) {
  const queryClient = useQueryClient();
  return useMutation<AdminIntakeEntry, Error, TVariables>({
    mutationFn,
    onSuccess: (entry) => {
      queryClient.setQueryData<AdminIntakeEntry[]>(adminIntakeCatalogKeys.list(), (entries) =>
        replaceEntry(entries, entry),
      );
      queryClient.invalidateQueries({ queryKey: [...adminIntakeCatalogKeys.all, "preview", entry.key] });
    },
  });
}

export function useCreateAdminIntake() {
  return useEntryMutation((input: CreateAdminIntakeInput) =>
    requestJSON<AdminIntakeEntry>(BASE_PATH, { method: "POST", body: JSON.stringify(input) }),
  );
}

export function useUpdateAdminIntake(key: string) {
  return useEntryMutation((input: UpdateAdminIntakeInput) =>
    requestJSON<AdminIntakeEntry>(entryPath(key), { method: "PATCH", body: JSON.stringify(input) }),
  );
}

export function useAddAdminIntakeOrganization(key: string) {
  return useEntryMutation((organizationId: string) =>
    requestJSON<AdminIntakeEntry>(`${entryPath(key)}/organizations/${encodeURIComponent(organizationId)}`, {
      method: "POST",
    }),
  );
}

export function useRemoveAdminIntakeOrganization(key: string) {
  return useEntryMutation((organizationId: string) =>
    requestJSON<AdminIntakeEntry>(`${entryPath(key)}/organizations/${encodeURIComponent(organizationId)}`, {
      method: "DELETE",
    }),
  );
}

export function useDeleteAdminIntake(key: string) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, void>({
    mutationFn: () => requestJSON<void>(entryPath(key), { method: "DELETE" }),
    onSuccess: () => {
      queryClient.setQueryData<AdminIntakeEntry[]>(adminIntakeCatalogKeys.list(), (entries) =>
        (entries ?? []).filter((entry) => entry.key !== key),
      );
    },
  });
}

export interface AdminOrganizationOption {
  id: string;
  name: string;
}

export function useAdminOrganizationSearch(search: string, enabled: boolean) {
  return useQuery({
    queryKey: ["adminIntakeCatalog", "organizations", search],
    queryFn: async () => {
      const params = new URLSearchParams({ limit: "20", offset: "0", sort_by: "name", sort_direction: "asc" });
      if (search) params.set("search", search);
      const data = await requestJSON<{ items?: AdminOrganizationOption[] }>(`/admin/api/organizations?${params}`);
      return (data.items ?? []).map((item) => ({ id: item.id, name: item.name }));
    },
    enabled,
    staleTime: 30 * 1000,
  });
}
