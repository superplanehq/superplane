import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { BillingPlans } from "./BillingPlans";
import { Features } from "./Features";

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn() }));
afterEach(() => vi.unstubAllGlobals());

it("creates a per-seat paid plan with dollar amounts and a markup percentage", async () => {
  const requests: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
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
  expect(screen.queryByLabelText("Included credit (USD)", { exact: true })).not.toBeInTheDocument();
  expect(screen.queryByLabelText("Credit validity (days)", { exact: true })).not.toBeInTheDocument();
  expect(screen.queryByRole("combobox", { name: "Credit frequency" })).not.toBeInTheDocument();
  expect(screen.queryByRole("combobox", { name: "Credit expiration" })).not.toBeInTheDocument();
  const trialCredit = screen.getByLabelText("Trial credit (USD)", { exact: true });
  await user.clear(trialCredit);
  await user.type(trialCredit, "0.29");
  await user.click(screen.getByRole("switch", { name: "Per seat" }));
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
        require_payment_method: true,
        features: ["byo_key"],
      }),
    ]),
  );
});

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
});

it("edits a linked plan in a dialog, preserves locked pricing, and saves unlimited limits", async () => {
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
      if (init?.method === "PUT") {
        updates.push({ path: String(input), body: JSON.parse(String(init.body)) });
        return new Response("{}");
      }
      return new Response(JSON.stringify(String(input).endsWith("feature-definitions") ? [] : [plan]));
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
  expect(screen.getByLabelText("Polar product ID", { exact: true })).toBeDisabled();
  await user.hover(screen.getByRole("button", { name: "Help for Seat limit" }));
  expect(await screen.findByRole("tooltip")).toHaveTextContent("Leave empty for unlimited seats.");
  await user.clear(screen.getByLabelText("Seat limit", { exact: true }));
  await user.clear(screen.getByLabelText("Parallel task limit", { exact: true }));
  await user.type(screen.getByLabelText("Parallel task limit", { exact: true }), "2");
  // Happy DOM incorrectly rejects decimal steps, so submit the form directly.
  fireEvent.submit(screen.getByRole("button", { name: "Save plan" }).closest("form")!);
  await waitFor(() =>
    expect(updates).toEqual([
      {
        path: "/admin/api/billing-plans/starter-plan",
        body: { ...plan, seat_limit: null, parallel_task_limit: 2, markup_bps: null },
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
  expect(screen.getByLabelText("Price (cents)", { exact: true })).toBeDisabled();
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
  expect(screen.queryByRole("button", { name: "Delete Business Manual" }) === null).toBe(true);
  await user.click(edit);
  expect(screen.getByRole("combobox", { name: "Category" }).hasAttribute("disabled")).toBe(true);
  expect(screen.getByLabelText("Polar product ID", { exact: true }).hasAttribute("disabled")).toBe(true);
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
