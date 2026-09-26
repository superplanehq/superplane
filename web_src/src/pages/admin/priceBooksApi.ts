export type PriceBookVersion = {
  version: string;
  effective_at: string;
  created_at: string;
};

export type PriceBookModelRate = {
  provider: string;
  match_key: string;
  match_mode: string;
  input_cents_per_million: number;
  output_cents_per_million: number;
  cache_read_cents_per_million: number;
  cache_write_cents_per_million: number;
  reasoning_cents_per_million: number;
  selected?: boolean;
};

export type PriceBookVMRate = {
  match_key: string;
  match_mode: string;
  micros_per_second: number;
};

export type PriceBooksResponse = {
  current_version: string;
  version: string;
  effective_at: string;
  created_at: string;
  versions: PriceBookVersion[];
  models: PriceBookModelRate[];
  vms: PriceBookVMRate[];
};

export type PriceBookSyncResponse = PriceBooksResponse & {
  updated_count: number;
  added_count: number;
  skipped_providers: string[];
};

async function readAdminError(response: Response, fallback: string): Promise<string> {
  const text = (await response.text()).trim();
  return text || fallback;
}

export async function fetchPriceBooks(version?: string, signal?: AbortSignal): Promise<PriceBooksResponse> {
  const path = version ? `/admin/api/price-books?version=${encodeURIComponent(version)}` : "/admin/api/price-books";
  const response = await fetch(path, { credentials: "include", signal });
  if (!response.ok) {
    throw new Error(await readAdminError(response, "Failed to load price books"));
  }

  return response.json();
}

export async function savePriceBooks(
  baseVersion: string,
  models: PriceBookModelRate[],
  vms: PriceBookVMRate[],
): Promise<PriceBooksResponse> {
  const response = await fetch("/admin/api/price-books", {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ base_version: baseVersion, models, vms }),
  });
  if (!response.ok) {
    throw new Error(await readAdminError(response, "Failed to save price books"));
  }

  return response.json();
}

export async function syncPriceBooks(provider = "openrouter"): Promise<PriceBookSyncResponse> {
  const response = await fetch(`/admin/api/price-books/sync?provider=${encodeURIComponent(provider)}`, {
    method: "POST",
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(await readAdminError(response, "Failed to update model rates"));
  }

  return response.json();
}

export async function deletePriceBook(version: string): Promise<PriceBooksResponse> {
  const response = await fetch(`/admin/api/price-books?version=${encodeURIComponent(version)}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(await readAdminError(response, "Failed to delete price book"));
  }

  return response.json();
}

export async function activatePriceBook(version: string): Promise<PriceBooksResponse> {
  const response = await fetch("/admin/api/price-books/current", {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ version }),
  });
  if (!response.ok) {
    throw new Error(await readAdminError(response, "Failed to switch price book"));
  }

  return response.json();
}
