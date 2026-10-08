import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import OwnerSetup from "./OwnerSetup";
import { OWNER_SETUP_COPY } from "./ownerSetup/ownerSetupCopy";

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });

vi.mock("./ownerSetup/LicenseStep", () => ({
  LicenseStep: ({ onContinue }: { onContinue: () => void }) => (
    <button type="button" data-testid="owner-setup-license-skip" onClick={onContinue}>
      Continue with Community
    </button>
  ),
}));

vi.mock("@/posthog", () => ({
  isPostHogEnabled: false,
  posthog: { getActiveMatchingSurveys: vi.fn() },
}));

describe("OwnerSetup", () => {
  const originalLocation = window.location;

  beforeEach(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, href: "http://localhost/setup" },
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", { configurable: true, value: originalLocation });
    vi.unstubAllGlobals();
  });

  it("reaches workspace setup when SMTP and Fleet Manager are skipped", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.endsWith("/api/v1/setup-owner") && init?.method === "POST") {
          return jsonResponse({ organization_id: "org-1", organization_slug: "acme" });
        }
        if (url.endsWith("/admin/api/installation/first-run/fleet-manager")) {
          return jsonResponse({ yaml: "id: self-host\n", token: "tok" });
        }
        return new Response("", { status: 404 });
      }),
    );

    render(
      <MemoryRouter>
        <OwnerSetup />
      </MemoryRouter>,
    );

    fireEvent.change(screen.getByPlaceholderText("First name"), { target: { value: "Ada" } });
    fireEvent.change(screen.getByPlaceholderText("Last name"), { target: { value: "Lovelace" } });
    fireEvent.change(screen.getByPlaceholderText("you@example.com"), { target: { value: "ada@example.com" } });
    fireEvent.change(screen.getByPlaceholderText("Password"), { target: { value: "Secret1x" } });
    fireEvent.change(screen.getByPlaceholderText("Confirm password"), { target: { value: "Secret1x" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByTestId("owner-setup-license-skip")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("owner-setup-license-skip"));

    expect(await screen.findByTestId("owner-setup-smtp")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("owner-setup-smtp-skip"));

    expect(await screen.findByTestId("owner-setup-fleet")).toBeInTheDocument();
    await screen.findByTestId("owner-setup-fleet-yaml");
    fireEvent.click(screen.getByTestId("owner-setup-fleet-skip"));

    await waitFor(() => expect(window.location.href).toBe("/acme/workspaces/new"));
    expect(screen.queryByText(OWNER_SETUP_COPY.smtp.headline)).not.toBeInTheDocument();
  });
});
