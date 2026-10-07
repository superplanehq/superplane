import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { AdminIntakeEntry } from "@/pages/admin/intakes/intakeCatalogModel";

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
}

export const adminIntakeCatalogKeys = {
  all: ["adminIntakeCatalog"] as const,
  list: () => [...adminIntakeCatalogKeys.all, "list"] as const,
  entry: (key: string) => [...adminIntakeCatalogKeys.all, "entry", key] as const,
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

async function fetchAdminIntakeEntry(key: string): Promise<AdminIntakeEntry> {
  return requestJSON<AdminIntakeEntry>(entryPath(key));
}

export function useAdminIntakeCatalogEntry(key: string) {
  return useQuery({
    queryKey: adminIntakeCatalogKeys.entry(key),
    queryFn: () => fetchAdminIntakeEntry(key),
    enabled: Boolean(key),
    staleTime: 30 * 1000,
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
      queryClient.setQueryData<AdminIntakeEntry>(adminIntakeCatalogKeys.entry(entry.key), entry);
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

export function useDeleteAdminIntake(key: string) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, void>({
    mutationFn: () => requestJSON<void>(entryPath(key), { method: "DELETE" }),
    onSuccess: () => {
      queryClient.setQueryData<AdminIntakeEntry[]>(adminIntakeCatalogKeys.list(), (entries) =>
        (entries ?? []).filter((entry) => entry.key !== key),
      );
      queryClient.removeQueries({ queryKey: adminIntakeCatalogKeys.entry(key) });
    },
  });
}
