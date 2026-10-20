import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { BillingPlans } from "./BillingPlans";

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn() }));
afterEach(() => vi.unstubAllGlobals());

it("creates a paid plan with no base fee and explains deferred credit delivery", async () => {
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
  await user.click(screen.getByRole("checkbox", { name: "Require payment method" }));
  await user.click(await screen.findByRole("checkbox", { name: /Bring your own API key/ }));
  expect(screen.getByText(/Other credit policies are stored for future delivery/)).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Save plan" }));
  await waitFor(() =>
    expect(requests).toEqual([
      expect.objectContaining({
        key: "payg",
        category: "paid",
        base_price_cents: 0,
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
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await screen.findByRole("tab", { name: /Plans/ });
  expect(screen.queryByRole("button", { name: "Create feature" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("tab", { name: /Features/ }));
  await user.click(await screen.findByRole("button", { name: "Create feature" }));
  expect(screen.getByRole("dialog", { name: "Create feature" })).toBeInTheDocument();
  await user.type(screen.getByLabelText("Feature key"), "custom_reports");
  await user.type(screen.getByLabelText("Feature name"), "Custom reports");
  await user.click(screen.getByRole("button", { name: "Save feature" }));
  await screen.findByRole("button", { name: "Edit Custom reports" });
  expect(definitions).toEqual([{ key: "custom_reports", label: "Custom reports", gated: true }]);
  await user.click(screen.getByRole("tab", { name: /Plans/ }));
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
    base_price_cents: 0,
    seat_price_cents: 500,
    polar_product_id: "polar-starter",
    seat_limit: 10,
    parallel_task_limit: 5,
    credit_amount_cents: 0,
    credit_frequency: "none",
    credit_expiration: "never",
    credit_validity_days: null,
    trial_credit_cents: 0,
    trial_length_days: 14,
    markup_bps: null,
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
  expect(planRow.getByText("$0.00 base")).toBeInTheDocument();
  expect(planRow.getByText("$5.00 / seat")).toBeInTheDocument();
  await user.click(editButton);
  expect(screen.getByRole("dialog", { name: "Edit Starter" })).toBeInTheDocument();
  expect(screen.getByLabelText("Key", { exact: true })).toBeDisabled();
  expect(screen.getByLabelText("Base price (cents)", { exact: true })).toBeDisabled();
  expect(screen.getByLabelText("Seat price (cents)", { exact: true })).toBeDisabled();
  expect(screen.getByLabelText("Polar product ID", { exact: true })).toBeDisabled();
  await user.hover(screen.getByRole("button", { name: "Help for Seat limit" }));
  expect(await screen.findByRole("tooltip")).toHaveTextContent("Leave empty for unlimited seats.");
  await user.clear(screen.getByLabelText("Seat limit", { exact: true }));
  await user.clear(screen.getByLabelText("Parallel task limit", { exact: true }));
  await user.type(screen.getByLabelText("Parallel task limit", { exact: true }), "2");
  await user.click(screen.getByRole("button", { name: "Save plan" }));
  await waitFor(() =>
    expect(updates).toEqual([
      {
        path: "/admin/api/billing-plans/starter-plan",
        body: { ...plan, seat_limit: null, parallel_task_limit: 2 },
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
