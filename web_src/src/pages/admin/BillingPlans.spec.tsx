import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { showErrorToast } from "@/lib/toast";
import { BillingPlans } from "./BillingPlans";
import type { BillingPlanListItem } from "./billingPlansApi";
import { Features } from "./Features";

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn() }));
afterEach(() => vi.unstubAllGlobals());

const systemReason = "System plans cannot be deleted.";
const polarReason = "This plan is linked to a Polar product and cannot be deleted.";
const assignedReason = "One or more organizations use this plan. Assign them another plan before deleting it.";

function listedPlan(overrides: Partial<BillingPlanListItem>): BillingPlanListItem {
  return {
    id: "plan-id",
    key: "custom",
    name: "Custom",
    category: "paid",
    visibility: "private",
    require_payment_method: false,
    price_cents: 1000,
    per_seat: false,
    polar_product_id: null,
    seat_limit: null,
    parallel_task_limit: null,
    credit_amount_cents: 0,
    credit_frequency: "none",
    credit_expiration: "never",
    credit_validity_days: null,
    trial_credit_cents: 0,
    trial_length_days: 14,
    markup_bps: null,
    features: [],
    delete_disabled_reason: null,
    ...overrides,
  };
}

type ListedPlan = BillingPlanListItem;

// stubBillingCatalog serves the list from mutable state. Requests record the
// method, path, and parsed body for assertions about what the page sent.
function stubBillingCatalog(
  initial: ListedPlan[],
  options: { onDelete?: (plan: ListedPlan, plans: ListedPlan[]) => Response } = {},
) {
  const requests: { method: string; path: string; body?: unknown }[] = [];
  let plans = initial;
  let failNextRefresh = false;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      requests.push({ method, path, body: init?.body ? JSON.parse(String(init.body)) : undefined });
      if (path.endsWith("feature-definitions")) return new Response("[]");
      if (method === "GET" && path.endsWith("billing-plans")) {
        if (failNextRefresh) {
          failNextRefresh = false;
          return new Response("Refresh failed", { status: 500 });
        }
        return new Response(JSON.stringify(plans));
      }
      if (method === "DELETE") {
        const plan = plans.find((candidate) => path.endsWith(`/${candidate.id}`));
        if (!plan) return new Response("Not found", { status: 404 });
        const response = options.onDelete?.(plan, plans) ?? new Response(null, { status: 204 });
        if (response.ok) plans = plans.filter((candidate) => candidate !== plan);
        return response;
      }
      return new Response("{}");
    }),
  );
  return { requests, failNextRefresh: () => (failNextRefresh = true) };
}

function deleteRequests(requests: { method: string }[]) {
  return requests.filter((request) => request.method === "DELETE");
}

it("creates a per-seat paid plan with dollar amounts and a markup percentage", async () => {
  const requests: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input) === "/admin/api/polar/products?kind=subscription") {
        return new Response(JSON.stringify([{ id: "prod_payg", name: "Pay as you go subscription" }]));
      }
      if (init?.method === "POST") {
        requests.push(JSON.parse(String(init.body)));
        return new Response("{}");
      }
      return new Response(
        JSON.stringify(
          String(input).endsWith("feature-definitions")
            ? [{ key: "byo_key", label: "Bring your own API key", gated: false }]
            : [],
        ),
      );
    }),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await user.click(screen.getByRole("button", { name: "Create plan" }));
  expect(screen.getByRole("dialog", { name: "Create plan" })).toBeInTheDocument();
  await user.type(screen.getByLabelText("Key"), "payg");
  await user.type(screen.getByLabelText("Name"), "Pay as you go");
  const price = screen.getByLabelText("Price (USD)", { exact: true });
  await user.clear(price);
  await user.type(price, "30.29");
  await user.type(screen.getByLabelText("Markup (%)", { exact: true }), "12.34");
  await user.click(screen.getByRole("combobox", { name: "Polar product" }));
  await user.click(await screen.findByRole("option", { name: /Pay as you go subscription/ }));
  expect(screen.queryByLabelText("Included credit (USD)", { exact: true })).not.toBeInTheDocument();
  expect(screen.queryByLabelText("Credit validity (days)", { exact: true })).not.toBeInTheDocument();
  expect(screen.queryByRole("combobox", { name: "Credit frequency" })).not.toBeInTheDocument();
  expect(screen.queryByRole("combobox", { name: "Credit expiration" })).not.toBeInTheDocument();
  const trialCredit = screen.getByLabelText("Trial credit (USD)", { exact: true });
  await user.clear(trialCredit);
  await user.type(trialCredit, "0.29");
  await user.click(screen.getByRole("switch", { name: "Per seat" }));
  expect(screen.getByLabelText("Maximum seats", { exact: true })).toHaveValue(50);
  await user.click(screen.getByRole("checkbox", { name: "Require payment method" }));
  await user.click(await screen.findByRole("checkbox", { name: /Bring your own API key/ }));
  // Happy DOM incorrectly rejects decimal steps, so submit the form directly.
  fireEvent.submit(screen.getByRole("button", { name: "Save plan" }).closest("form")!);
  await waitFor(() =>
    expect(requests).toEqual([
      expect.objectContaining({
        key: "payg",
        category: "paid",
        price_cents: 3029,
        markup_bps: 1234,
        credit_amount_cents: 0,
        credit_frequency: "none",
        credit_expiration: "never",
        credit_validity_days: null,
        trial_credit_cents: 29,
        per_seat: true,
        seat_limit: 50,
        require_payment_method: true,
        polar_product_id: "prod_payg",
        features: ["byo_key"],
      }),
    ]),
  );
}, 15000);

it("creates a feature and selects its string key in a plan", async () => {
  const definitions: { key: string; label: string; gated: boolean }[] = [];
  const savedPlans: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const isFeature = String(input).endsWith("feature-definitions");
      if (init?.method === "POST") {
        const body = JSON.parse(String(init.body));
        if (isFeature) definitions.push(body);
        else savedPlans.push(body);
        return new Response(JSON.stringify(body));
      }
      return new Response(JSON.stringify(isFeature ? definitions : []));
    }),
  );
  const user = userEvent.setup();
  const view = render(
    <MemoryRouter>
      <Features />
    </MemoryRouter>,
  );
  await user.click(await screen.findByRole("button", { name: "Create feature" }));
  expect(screen.getByRole("dialog", { name: "Create feature" })).toBeInTheDocument();
  await user.type(screen.getByLabelText("Feature key"), "custom_reports");
  await user.type(screen.getByLabelText("Feature name"), "Custom reports");
  await user.click(screen.getByRole("button", { name: "Save feature" }));
  await screen.findByRole("button", { name: "Edit Custom reports" });
  expect(definitions).toEqual([{ key: "custom_reports", label: "Custom reports", gated: true }]);
  view.unmount();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await waitFor(() => expect(screen.getByRole("button", { name: "Create plan" })).toBeEnabled());
  expect(screen.queryByRole("tab", { name: /Features/ })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Create plan" }));
  await user.type(screen.getByLabelText("Key"), "reports");
  await user.type(screen.getByLabelText("Name"), "Reports");
  await user.click(screen.getByRole("checkbox", { name: /Custom reports/ }));
  await user.click(screen.getByRole("button", { name: "Save plan" }));
  await waitFor(() => expect(savedPlans).toEqual([expect.objectContaining({ features: ["custom_reports"] })]));
}, 15000);

it("edits a linked plan in a dialog, preserves locked pricing and credit policies, and updates maximum seats", async () => {
  const plan = {
    id: "starter-plan",
    key: "starter",
    name: "Starter",
    category: "paid",
    visibility: "public",
    require_payment_method: true,
    price_cents: 599,
    per_seat: true,
    polar_product_id: "polar-starter",
    seat_limit: 10,
    parallel_task_limit: 5,
    credit_amount_cents: 1234,
    credit_frequency: "billing_period",
    credit_expiration: "after_days",
    credit_validity_days: 30,
    trial_credit_cents: 678,
    trial_length_days: 14,
    markup_bps: 1000,
    features: [],
  };
  const updates: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input) === "/admin/api/polar/products?kind=subscription") {
        return new Response(JSON.stringify([{ id: "polar-starter", name: "Starter subscription" }]));
      }
      if (init?.method === "PUT") {
        updates.push({ path: String(input), body: JSON.parse(String(init.body)) });
        return new Response("{}");
      }
      return new Response(
        JSON.stringify(
          String(input).endsWith("feature-definitions") ? [] : [{ ...plan, delete_disabled_reason: null }],
        ),
      );
    }),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  const editButton = await screen.findByRole("button", { name: "Edit Starter" });
  const planRow = within(screen.getByRole("row", { name: /Starter/ }));
  expect(planRow.getByText("$5.99 per seat")).toBeInTheDocument();
  await user.click(editButton);
  expect(screen.getByRole("dialog", { name: "Edit Starter" })).toBeInTheDocument();
  expect(screen.getByLabelText("Key", { exact: true })).toBeDisabled();
  expect(screen.getByLabelText("Price (USD)", { exact: true })).toBeDisabled();
  expect(screen.getByLabelText("Price (USD)", { exact: true })).toHaveValue(5.99);
  expect(screen.getByLabelText("Trial credit (USD)", { exact: true })).toHaveValue(6.78);
  expect(screen.getByLabelText("Markup (%)", { exact: true })).toHaveValue(10);
  await user.clear(screen.getByLabelText("Markup (%)", { exact: true }));
  expect(screen.getByRole("switch", { name: "Per seat" })).toBeDisabled();
  expect(screen.getByRole("combobox", { name: "Polar product" })).toBeDisabled();
  await waitFor(() =>
    expect(screen.getByRole("combobox", { name: "Polar product" })).toHaveTextContent("Starter subscription"),
  );
  const maximum = screen.getByLabelText("Maximum seats", { exact: true });
  expect(maximum).toHaveValue(10);
  await user.clear(maximum);
  await user.type(maximum, "75");
  await user.clear(screen.getByLabelText("Parallel task limit", { exact: true }));
  await user.type(screen.getByLabelText("Parallel task limit", { exact: true }), "2");
  // Happy DOM incorrectly rejects decimal steps, so submit the form directly.
  fireEvent.submit(screen.getByRole("button", { name: "Save plan" }).closest("form")!);
  await waitFor(() =>
    expect(updates).toEqual([
      {
        path: "/admin/api/billing-plans/starter-plan",
        body: { ...plan, seat_limit: 75, parallel_task_limit: 2, markup_bps: null },
      },
    ]),
  );
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await user.click(screen.getByRole("button", { name: "Edit Starter" }));
  await user.type(screen.getByLabelText("Name", { exact: true }), " changed");
  await user.click(screen.getByRole("button", { name: "Cancel" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "Edit Starter" })).toHaveFocus());
  expect(updates).toHaveLength(1);
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});

it("locks legacy Business pricing and usage limits in the editor", async () => {
  const plan = {
    id: "legacy-business",
    key: "business_legacy",
    name: "Business",
    category: "paid",
    visibility: "private",
    require_payment_method: false,
    price_cents: 19900,
    per_seat: false,
    polar_product_id: null,
    seat_limit: null,
    parallel_task_limit: null,
    credit_amount_cents: 5000,
    credit_frequency: "billing_period",
    credit_expiration: "period_end",
    credit_validity_days: null,
    trial_credit_cents: 0,
    trial_length_days: 14,
    markup_bps: null,
    features: [],
  };
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async (input: RequestInfo | URL) =>
        new Response(JSON.stringify(String(input).endsWith("feature-definitions") ? [] : [plan])),
    ),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await user.click(await screen.findByRole("button", { name: "Edit Business" }));
  expect(screen.getByLabelText("Price (USD)", { exact: true })).toBeDisabled();
  expect(screen.getByLabelText("Seat limit", { exact: true })).toBeDisabled();
  expect(screen.getByLabelText("Parallel task limit", { exact: true })).toBeDisabled();
});

it("protects Business Manual from deletion and unsupported catalog edits", async () => {
  const plan = {
    id: "manual-business",
    key: "business_manual",
    name: "Business Manual",
    category: "paid",
    visibility: "private",
    price_cents: 19900,
    per_seat: false,
    polar_product_id: null,
    credit_frequency: "billing_period",
    credit_expiration: "period_end",
    features: [],
    delete_disabled_reason: systemReason,
  };
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async (input: RequestInfo | URL) =>
        new Response(JSON.stringify(String(input).endsWith("feature-definitions") ? [] : [plan])),
    ),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  const edit = await screen.findByRole("button", { name: "Edit Business Manual" });
  expect(screen.getByRole("button", { name: "Delete Business Manual" })).toBeDisabled();
  await user.hover(screen.getByRole("group", { name: "Delete Business Manual is unavailable" }));
  expect(await screen.findByRole("tooltip")).toHaveTextContent(systemReason);
  await user.click(edit);
  expect(screen.getByRole("combobox", { name: "Category" }).hasAttribute("disabled")).toBe(true);
  expect(screen.getByRole("combobox", { name: "Polar product" })).toBeDisabled();
});

it("links the empty plan feature selector to Features", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response("[]")),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await waitFor(() => expect(screen.getByRole("button", { name: "Create plan" })).toBeEnabled());
  await user.click(screen.getByRole("button", { name: "Create plan" }));
  expect(screen.getByRole("link", { name: "Features page" })).toHaveAttribute("href", "/admin/features");
});

it("shows every blocked delete action with its reason on hover and keyboard focus", async () => {
  const { requests } = stubBillingCatalog([
    listedPlan({ id: "system", key: "none", name: "Free", delete_disabled_reason: systemReason }),
    listedPlan({
      id: "polar",
      key: "polar_plan",
      name: "Polar",
      polar_product_id: "prod_1",
      delete_disabled_reason: polarReason,
    }),
    listedPlan({ id: "assigned", key: "assigned_plan", name: "Assigned", delete_disabled_reason: assignedReason }),
  ]);
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  const assignedDelete = await screen.findByRole("button", { name: "Delete Assigned" });
  for (const name of ["Delete Free", "Delete Polar", "Delete Assigned"]) {
    expect(screen.getByRole("button", { name })).toBeDisabled();
  }

  await user.hover(screen.getByRole("group", { name: "Delete Assigned is unavailable" }));
  expect(await screen.findByRole("tooltip")).toHaveTextContent(assignedReason);
  await user.unhover(screen.getByRole("group", { name: "Delete Assigned is unavailable" }));

  act(() => screen.getByRole("button", { name: "Edit Assigned" }).focus());
  await user.tab();
  expect(screen.getByRole("group", { name: "Delete Assigned is unavailable" })).toHaveFocus();
  expect(await screen.findByRole("tooltip")).toHaveTextContent(assignedReason);

  await user.keyboard("{Enter}");
  await user.keyboard(" ");
  await user.click(assignedDelete);
  expect(deleteRequests(requests)).toEqual([]);
});

it("explains a Polar-linked plan that cannot be deleted on hover", async () => {
  stubBillingCatalog([
    listedPlan({
      id: "polar",
      key: "polar_plan",
      name: "Polar",
      polar_product_id: "prod_1",
      delete_disabled_reason: polarReason,
    }),
  ]);
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await user.hover(await screen.findByRole("group", { name: "Delete Polar is unavailable" }));
  expect(await screen.findByRole("tooltip")).toHaveTextContent(polarReason);
});

it("deletes an enabled plan and removes it after the list reloads", async () => {
  const { requests } = stubBillingCatalog([listedPlan({ id: "spare", key: "spare", name: "Spare" })]);
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await user.click(await screen.findByRole("button", { name: "Delete Spare" }));
  await waitFor(() => expect(screen.queryByText("Spare")).not.toBeInTheDocument());
  expect(deleteRequests(requests)).toEqual([{ method: "DELETE", path: "/admin/api/billing-plans/spare" }]);
});

it("shows a rejected delete, reloads, and disables the newly assigned plan", async () => {
  const { requests } = stubBillingCatalog([listedPlan({ id: "stale", key: "stale", name: "Stale" })], {
    onDelete: (plan, plans) => {
      plans.splice(0, plans.length, { ...plan, delete_disabled_reason: assignedReason });
      return new Response(assignedReason, { status: 400 });
    },
  });
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await user.click(await screen.findByRole("button", { name: "Delete Stale" }));
  await waitFor(() => expect(showErrorToast).toHaveBeenCalledWith(assignedReason));
  expect(await screen.findByRole("button", { name: "Delete Stale" })).toBeDisabled();
  expect(deleteRequests(requests)).toHaveLength(1);
  expect(requests.filter((request) => request.method === "GET" && request.path.endsWith("billing-plans"))).toHaveLength(
    2,
  );
});

it("keeps the delete error when the reload fails and recovers with Try again", async () => {
  const catalog = stubBillingCatalog([listedPlan({ id: "stale", key: "stale", name: "Stale" })], {
    onDelete: (plan, plans) => {
      plans.splice(0, plans.length, { ...plan, delete_disabled_reason: assignedReason });
      return new Response(assignedReason, { status: 400 });
    },
  });
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  const deleteButton = await screen.findByRole("button", { name: "Delete Stale" });
  catalog.failNextRefresh();
  await user.click(deleteButton);
  await waitFor(() => expect(showErrorToast).toHaveBeenCalledWith(assignedReason));
  const alert = await screen.findByRole("alert");
  expect(alert).toHaveTextContent("Refresh failed");
  await user.click(within(alert).getByRole("button", { name: "Try again" }));
  expect(await screen.findByRole("button", { name: "Delete Stale" })).toBeDisabled();
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});
