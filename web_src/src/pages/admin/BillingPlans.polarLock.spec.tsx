import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { BillingPlans } from "./BillingPlans";

afterEach(() => vi.unstubAllGlobals());

it("unlocks the Polar product, price, and Per seat on a linked plan that no organization uses", async () => {
  const plan = {
    id: "unused-plan",
    key: "unused",
    name: "Unused",
    category: "paid",
    visibility: "private",
    require_payment_method: false,
    price_cents: 599,
    per_seat: true,
    polar_product_id: "polar-wrong",
    seat_limit: 10,
    parallel_task_limit: null,
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
      if (String(input) === "/admin/api/polar/products?kind=subscription") {
        return new Response(JSON.stringify([{ id: "polar-wrong", name: "Wrong subscription" }]));
      }
      if (init?.method === "PUT") {
        updates.push({ path: String(input), body: JSON.parse(String(init.body)) });
        return new Response("{}");
      }
      const listed = [{ ...plan, delete_disabled_reason: null, assigned: false }];
      return new Response(JSON.stringify(String(input).endsWith("feature-definitions") ? [] : listed));
    }),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await user.click(await screen.findByRole("button", { name: "Edit Unused" }));
  expect(screen.getByLabelText("Price (USD)", { exact: true })).toBeEnabled();
  expect(screen.getByRole("switch", { name: "Per seat" })).toBeEnabled();
  const product = screen.getByRole("combobox", { name: "Polar product" });
  await waitFor(() => expect(product).toBeEnabled());
  expect(screen.queryByText(/Create a new plan to change the product or price/)).not.toBeInTheDocument();
  await user.click(product);
  await user.click(await screen.findByRole("option", { name: "No product" }));
  // Happy DOM incorrectly rejects decimal steps, so submit the form directly.
  fireEvent.submit(screen.getByRole("button", { name: "Save plan" }).closest("form")!);
  await waitFor(() =>
    expect(updates).toEqual([
      { path: "/admin/api/billing-plans/unused-plan", body: { ...plan, polar_product_id: null } },
    ]),
  );
});
