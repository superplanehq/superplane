export type PriceBookVersion = {
  version: string;
  effective_at: string;
  created_at: string;
};

export type PriceBookModelRate = {
  match_key: string;
  match_mode: string;
  input_cents_per_million: number;
  output_cents_per_million: number;
  cache_read_cents_per_million: number;
  cache_write_cents_per_million: number;
  reasoning_cents_per_million: number;
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

export async function fetchPriceBooks(version?: string, signal?: AbortSignal): Promise<PriceBooksResponse> {
  const path = version ? `/admin/api/price-books?version=${encodeURIComponent(version)}` : "/admin/api/price-books";
  const response = await fetch(path, { credentials: "include", signal });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text.trim() || "Failed to load price books");
  }

  return response.json();
}
