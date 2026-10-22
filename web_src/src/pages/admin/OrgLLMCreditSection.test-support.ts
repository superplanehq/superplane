import { vi } from "bun:test";
import type { OrganizationLLMCredit } from "./useOrgLLMCredit";

const trialEnd = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();

export const credit: OrganizationLLMCredit = {
  remaining_credit_cents: 9500,
  grant_total_cents: 10000,
  superplane_grant_cents: 7000,
  purchased_credit_cents: 3000,
  hosted_billed_cents: 500,
  welcome_remaining_cents: 4500,
  included_remaining_cents: 0,
  purchased_remaining_cents: 3000,
  admin_remaining_cents: 2000,
  welcome_credit_expires_at: trialEnd,
  markup_bps: 2000,
  markup_override_bps: null,
  warning: false,
};

export const catalog = [
  { key: "trial", name: "Trial", category: "trial", require_payment_method: false },
  {
    key: "business_legacy",
    name: "Business",
    category: "paid",
    require_payment_method: true,
    polar_product_id: "prod_business",
    price_cents: 2500,
  },
  { key: "business_manual", name: "Business Manual", category: "paid", require_payment_method: false },
  { key: "pay_as_you_go", name: "Pay-as-you-go", category: "paid", require_payment_method: true },
  { key: "none", name: "None", category: "none", require_payment_method: false },
];

export const grants = [
  {
    id: "grant-2",
    kind: "trial_adjustment",
    amountCents: "-500",
    note: "Support correction",
    actorName: "Ada",
    createdAt: new Date().toISOString(),
  },
  { id: "grant-1", kind: "welcome", amountCents: "5000", note: "", createdAt: new Date().toISOString() },
];

export function billingPlan(
  polarManaged: boolean,
  trialEndsAt: string | null = null,
  plan = polarManaged ? "business" : "trial",
  catalogCategory?: string,
) {
  return {
    plan,
    catalog_key: plan === "business" ? (polarManaged ? "business_legacy" : "business_manual") : plan,
    catalog_category: catalogCategory ?? (plan === "trial" ? "trial" : plan === "none" ? "none" : "paid"),
    plan_source: polarManaged ? "polar" : plan === "business" ? "admin" : "system",
    polar_subscription_status: polarManaged ? "active" : "",
    polar_managed: polarManaged,
    polar_subscription_verification: polarManaged ? "active" : "not_configured",
    polar_subscription_id: "",
    polar_subscription_url: "",
    trial_ends_at: trialEndsAt,
    current_period_end: null,
  };
}

export function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

export function mockOrgCreditFetch(
  options: {
    polarManaged?: boolean;
    planStatus?: number;
    plan?: ReturnType<typeof billingPlan>;
    creditData?: typeof credit;
    onPutBalance?: (body: unknown) => Response;
    onGetGrants?: () => Response;
    catalogStatus?: number;
  } = {},
) {
  const {
    catalogStatus = 200,
    onPutBalance = () => jsonResponse(options.creditData ?? credit),
    onGetGrants = () => jsonResponse({ grants }),
  } = options;
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.endsWith("/feature-definitions")) return jsonResponse([]);
    if (url.endsWith("/features")) return jsonResponse({ grants: [], effective: { features: {} } });
    if (url.endsWith("/llm-credit/balances") && init?.method === "PUT") {
      const body = JSON.parse(String(init.body));
      return onPutBalance(body);
    }
    if (url.endsWith("/llm-credit/grants")) {
      return onGetGrants();
    }
    if (url.endsWith("/llm-credit")) {
      return jsonResponse(options.creditData ?? credit);
    }
    if (url.endsWith("/billing-plans")) {
      return jsonResponse(catalog, catalogStatus);
    }
    if (url.includes("/billing-plan")) {
      if (options.planStatus) {
        return new Response("failed", { status: options.planStatus });
      }
      return jsonResponse(options.plan ?? billingPlan(options.polarManaged ?? false));
    }
    return new Response("not found", { status: 404 });
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}
