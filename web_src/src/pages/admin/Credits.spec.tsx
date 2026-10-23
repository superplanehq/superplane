import { render, screen, waitFor } from "@testing-library/react";
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

afterEach(() => vi.unstubAllGlobals());

describe("Credits", () => {
  for (const tc of [
    { percent: "6.5", dollars: "0.40", bps: 650, cents: 40 },
    { percent: "0", dollars: "0", bps: 0, cents: 0 },
  ]) {
    it(`saves ${tc.percent}% and $${tc.dollars} without changing other settings`, async () => {
      const user = userEvent.setup();
      const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) =>
        jsonResponse(
          init?.method === "PATCH"
            ? { ...settings, credit_purchase_fee_bps: tc.bps, credit_purchase_fixed_fee_cents: tc.cents }
            : settings,
        ),
      );
      vi.stubGlobal("fetch", fetchMock);
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
    const fetchMock = vi.fn(async () => jsonResponse(settings));
    vi.stubGlobal("fetch", fetchMock);
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
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("keeps edits after a failed save and allows retry", async () => {
    const user = userEvent.setup();
    let saves = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method !== "PATCH") return jsonResponse(settings);
        saves += 1;
        if (saves === 1) return new Response("Fee settings could not be saved. Try again.", { status: 500 });
        return jsonResponse({ ...settings, credit_purchase_fee_bps: 600 });
      }),
    );
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
});
