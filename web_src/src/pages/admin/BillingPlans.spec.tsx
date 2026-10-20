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
  await user.click(await screen.findByRole("checkbox", { name: /Bring your own API key/ }));
  expect(screen.getByText(/Other credit policies are stored for future delivery/)).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Save plan" }));
  await waitFor(() =>
    expect(requests).toEqual([
      expect.objectContaining({ key: "payg", category: "paid", base_price_cents: 0, features: ["byo_key"] }),
    ]),
  );
});
