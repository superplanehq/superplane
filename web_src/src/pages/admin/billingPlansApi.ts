export type BillingPlan = {
  id?: string;
  key: string;
  name: string;
  category: string;
  visibility: string;
  require_payment_method: boolean;
  price_cents: number;
  per_seat: boolean;
  polar_product_id: string | null;
  seat_limit: number | null;
  parallel_task_limit: number | null;
  credit_amount_cents: number;
  credit_frequency: string;
  credit_expiration: string;
  credit_validity_days: number | null;
  trial_credit_cents: number;
  trial_length_days: number;
  markup_bps: number | null;
  features: string[];
};

// The admin list adds delete_disabled_reason and assigned. They are display
// metadata only and never belong in the editable plan or in POST and PUT payloads.
export type BillingPlanListItem = BillingPlan & { delete_disabled_reason: string | null; assigned: boolean };

export function toEditableBillingPlan({
  delete_disabled_reason: _reason,
  assigned: _assigned,
  ...plan
}: BillingPlanListItem): BillingPlan {
  return plan;
}

export type FeatureDefinition = { key: string; label: string; gated: boolean };
export type OrganizationFeatures = {
  grants: { feature_key: string; enabled: boolean }[];
  effective: { features: Record<string, boolean> };
};

export async function billingCatalogRequest<T>(path: string, method = "GET", body?: unknown): Promise<T> {
  const response = await fetch(`/admin/api/${path}`, {
    method,
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!response.ok) throw new Error((await response.text()) || "Failed to update billing catalog");
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
