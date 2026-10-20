import { render, screen, waitFor } from "@testing-library/react";
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
  await user.click(await screen.findByRole("button", { name: "Create feature" }));
  await user.type(screen.getByLabelText("Feature key"), "custom_reports");
  await user.type(screen.getByLabelText("Feature name"), "Custom reports");
  await user.click(screen.getByRole("button", { name: "Save feature" }));
  await screen.findByRole("button", { name: "Edit Custom reports" });
  expect(definitions).toEqual([{ key: "custom_reports", label: "Custom reports", gated: true }]);
  await user.click(screen.getByRole("button", { name: "Create plan" }));
  await user.type(screen.getByLabelText("Key"), "reports");
  await user.type(screen.getByLabelText("Name"), "Reports");
  await user.click(screen.getByRole("checkbox", { name: /Custom reports/ }));
  await user.click(screen.getByRole("button", { name: "Save plan" }));
  await waitFor(() => expect(savedPlans).toEqual([expect.objectContaining({ features: ["custom_reports"] })]));
});
