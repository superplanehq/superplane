import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { Credits } from "./Credits";
import type { InstallationLLMSettings } from "./hostedLLMSettingsApi";

const settings: InstallationLLMSettings = {
  welcome_grant_cents: 5000,
  markup_bps: 1000,
  warning_threshold_bps: 2000,
  credit_purchase_fee_bps: 500,
  credit_purchase_fixed_fee_cents: 50,
  providers: [],
};

const renderCredits = () =>
  render(
    <MemoryRouter>
      <Credits />
    </MemoryRouter>,
  );
const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

function stubCreditsFetch(handler: (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>) {
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    if (String(input) === "/admin/api/credit-packs") return jsonResponse([]);
    return handler(input, init);
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

afterEach(() => vi.unstubAllGlobals());

describe("Credits", () => {
  for (const tc of [
    { percent: "6.5", dollars: "0.40", bps: 650, cents: 40 },
    { percent: "0", dollars: "0", bps: 0, cents: 0 },
  ]) {
    it(`saves ${tc.percent}% and $${tc.dollars} without changing other settings`, async () => {
      const user = userEvent.setup();
      const fetchMock = stubCreditsFetch(async (_input: RequestInfo | URL, init?: RequestInit) =>
        jsonResponse(
          init?.method === "PATCH"
            ? { ...settings, credit_purchase_fee_bps: tc.bps, credit_purchase_fixed_fee_cents: tc.cents }
            : settings,
        ),
      );
      renderCredits();

      const percent = await screen.findByLabelText("Processing fee percent");
      const fixed = screen.getByLabelText("Fixed fee per purchase (USD)");
      expect(percent).toHaveValue("5");
      expect(fixed).toHaveValue("0.50");
      expect(screen.getByRole("button", { name: "Save fee settings" })).toBeDisabled();

      await user.clear(percent);
      await user.type(percent, tc.percent);
      await user.clear(fixed);
      await user.type(fixed, tc.dollars);
      await user.click(screen.getByRole("button", { name: "Save fee settings" }));

      await waitFor(() =>
        expect(fetchMock).toHaveBeenCalledWith(
          "/admin/api/installation/llm-settings",
          expect.objectContaining({
            method: "PATCH",
            body: JSON.stringify({
              credit_purchase_fee_bps: tc.bps,
              credit_purchase_fixed_fee_cents: tc.cents,
            }),
          }),
        ),
      );
      expect(await screen.findByRole("button", { name: "Save fee settings" })).toBeDisabled();
    });
  }

  it("prevents invalid or empty input from silently becoming a zero fee", async () => {
    const user = userEvent.setup();
    const fetchMock = stubCreditsFetch(async () => jsonResponse(settings));
    renderCredits();
    const percent = await screen.findByLabelText("Processing fee percent");
    const fixed = screen.getByLabelText("Fixed fee per purchase (USD)");

    for (const value of ["", "-1", "100", "5.123", "invalid"]) {
      await user.clear(percent);
      if (value) await user.type(percent, value);
      expect(percent).toHaveAttribute("aria-invalid", "true");
      expect(screen.getByRole("button", { name: "Save fee settings" })).toBeDisabled();
    }
    await user.clear(percent);
    await user.type(percent, "5");
    for (const value of ["", "-1", "0.001", "invalid", "9999999999999999"]) {
      await user.clear(fixed);
      if (value) await user.type(fixed, value);
      expect(fixed).toHaveAttribute("aria-invalid", "true");
      expect(screen.getByRole("button", { name: "Save fee settings" })).toBeDisabled();
    }
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === "PATCH")).toHaveLength(0);
  });

  it("keeps edits after a failed save and allows retry", async () => {
    const user = userEvent.setup();
    let saves = 0;
    stubCreditsFetch(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method !== "PATCH") return jsonResponse(settings);
      saves += 1;
      if (saves === 1) return new Response("Fee settings could not be saved. Try again.", { status: 500 });
      return jsonResponse({ ...settings, credit_purchase_fee_bps: 600 });
    });
    renderCredits();
    const percent = await screen.findByLabelText("Processing fee percent");
    await user.clear(percent);
    await user.type(percent, "6");
    await user.click(screen.getByRole("button", { name: "Save fee settings" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Fee settings could not be saved. Try again.");
    expect(percent).toHaveValue("6");
    await user.click(screen.getByRole("button", { name: "Save fee settings" }));
    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
    expect(saves).toBe(2);
  });

  it("creates and edits a credit pack with dollar amounts and a Polar product", async () => {
    const user = userEvent.setup();
    const requests: { method: string; body: unknown }[] = [];
    let packs: unknown[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).includes("llm-settings")) return jsonResponse(settings);
        if (String(input).endsWith("polar/products"))
          return jsonResponse([{ id: "prod_starter", name: "Starter product" }]);
        const method = init?.method ?? "GET";
        if (method === "GET") return jsonResponse(packs);
        if (method === "DELETE") {
          requests.push({ method, body: undefined });
          packs = [];
          return new Response(null, { status: 204 });
        }
        const body = JSON.parse(String(init?.body));
        requests.push({ method, body });
        packs = [{ ...body, id: "pack-1" }];
        return jsonResponse(packs[0]);
      }),
    );
    renderCredits();
    await user.click(await screen.findByRole("button", { name: "Create pack" }));
    const dialog = within(screen.getByRole("dialog"));
    await user.type(dialog.getByLabelText("Name"), "Starter credits");
    await user.type(dialog.getByLabelText("Credit amount (USD)"), "25.50");
    await user.click(dialog.getByRole("combobox", { name: "Polar product" }));
    await user.click(await screen.findByRole("option", { name: /Starter product/ }));
    await user.click(dialog.getByRole("button", { name: "Save pack" }));
    expect(await screen.findByText("Starter credits")).toBeInTheDocument();
    expect(requests[0]).toEqual({
      method: "POST",
      body: {
        name: "Starter credits",
        amount_cents: 2550,
        enabled: true,
        polar_product_id: "prod_starter",
      },
    });
    await user.click(screen.getByRole("button", { name: "Edit Starter credits" }));
    const editDialog = within(screen.getByRole("dialog"));
    expect(editDialog.getByLabelText("Credit amount (USD)")).toHaveValue("25.50");
    await waitFor(() =>
      expect(editDialog.getByRole("combobox", { name: "Polar product" })).toHaveTextContent("Starter product"),
    );
    await user.click(editDialog.getByRole("switch", { name: "Enabled" }));
    await user.click(editDialog.getByRole("button", { name: "Save pack" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(requests[1]).toEqual({
      method: "PUT",
      body: {
        name: "Starter credits",
        amount_cents: 2550,
        enabled: false,
        polar_product_id: "prod_starter",
      },
    });
    expect(screen.getByText("Disabled")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete Starter credits" }));
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Keep pack" }));
    expect(screen.getByText("Starter credits")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete Starter credits" }));
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Delete pack" }));
    await waitFor(() => expect(screen.queryByText("Starter credits")).not.toBeInTheDocument());
    expect(requests[2]).toEqual({ method: "DELETE", body: undefined });
  });

  it("saves a custom amount pack before linking a Polar product", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).includes("llm-settings")) return jsonResponse(settings);
      if (init?.method === "POST") return jsonResponse({ id: "custom-pack" });
      return jsonResponse([]);
    });
    vi.stubGlobal("fetch", fetchMock);
    renderCredits();
    await user.click(await screen.findByRole("button", { name: "Create pack" }));
    const dialog = within(screen.getByRole("dialog"));
    await user.type(dialog.getByLabelText("Name"), "Custom credits");
    await user.click(dialog.getByRole("switch", { name: "Custom amount" }));
    expect(dialog.queryByLabelText("Credit amount (USD)")).not.toBeInTheDocument();
    await user.click(dialog.getByRole("button", { name: "Save pack" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(fetchMock).toHaveBeenCalledWith(
      "/admin/api/credit-packs",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ name: "Custom credits", amount_cents: 0, enabled: true, polar_product_id: null }),
      }),
    );
  });

  it("keeps pack edits after a failed save and validates the credit amount", async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).includes("llm-settings")) return jsonResponse(settings);
        if (init?.method === "POST") return new Response("This Polar product is already linked.", { status: 400 });
        return jsonResponse([]);
      }),
    );
    renderCredits();
    await user.click(await screen.findByRole("button", { name: "Create pack" }));
    const dialog = within(screen.getByRole("dialog"));
    await user.type(dialog.getByLabelText("Name"), "New credits");
    const amount = dialog.getByLabelText("Credit amount (USD)");
    for (const value of ["", "0", "-1", "1.234", "1000000", "invalid"]) {
      await user.clear(amount);
      if (value) await user.type(amount, value);
      expect(dialog.getByRole("button", { name: "Save pack" })).toBeDisabled();
    }
    await user.clear(amount);
    await user.type(amount, "50");
    await user.click(dialog.getByRole("button", { name: "Save pack" }));
    expect(await dialog.findByRole("alert")).toHaveTextContent("This Polar product is already linked.");
    expect(amount).toHaveValue("50");
    expect(dialog.getByLabelText("Name")).toHaveValue("New credits");
  });

  it("preserves the saved Polar product on load failure and allows retry and unlinking", async () => {
    const user = userEvent.setup();
    let productLoads = 0;
    const savedPack = {
      id: "saved",
      name: "Saved credits",
      amount_cents: 2500,
      enabled: true,
      polar_product_id: "prod_saved",
    };
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).includes("llm-settings")) return jsonResponse(settings);
      if (String(input).endsWith("polar/products")) {
        productLoads++;
        if (productLoads === 1) return new Response("Could not load Polar products. Try again.", { status: 502 });
        return jsonResponse([{ id: "prod_other", name: "Another product" }]);
      }
      if (init?.method === "PUT") return jsonResponse(savedPack);
      return jsonResponse([savedPack]);
    });
    vi.stubGlobal("fetch", fetchMock);
    renderCredits();
    await user.click(await screen.findByRole("button", { name: "Edit Saved credits" }));
    const dialog = within(screen.getByRole("dialog"));
    expect(await dialog.findByRole("alert")).toHaveTextContent("Could not load Polar products");
    expect(dialog.getByText("prod_saved")).toBeInTheDocument();
    expect(dialog.getByRole("button", { name: "Save pack" })).toBeEnabled();
    await user.click(dialog.getByRole("button", { name: "Retry products" }));
    await waitFor(() => expect(dialog.queryByRole("alert")).not.toBeInTheDocument());
    expect(dialog.getByText("prod_saved")).toBeInTheDocument();
    await user.click(dialog.getByRole("combobox", { name: "Polar product" }));
    await user.click(screen.getByRole("option", { name: "No product" }));
    await user.click(dialog.getByRole("button", { name: "Save pack" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(fetchMock).toHaveBeenCalledWith(
      "/admin/api/credit-packs/saved",
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ name: "Saved credits", amount_cents: 2500, enabled: true, polar_product_id: null }),
      }),
    );
  });
});
