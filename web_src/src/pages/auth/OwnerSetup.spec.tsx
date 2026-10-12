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

const ONBOARDING_LABELS = ["Discover", "Awaiting code", "Verify", "01 Plan", "02 Build", "03 Check", "04 Review"];

function expectOwnerSetupPreview() {
  const root = screen.getByTestId("owner-setup");
  expect(root).toHaveAttribute("data-visual", "preview");
  expect(screen.getByTestId("first-run-logo")).toHaveAttribute("alt", "SuperPlane");
  expect(root.querySelector("canvas")).toBeNull();
  expect(screen.queryByTestId("owner-setup-pane")).not.toBeInTheDocument();
  expect(screen.queryByTestId("first-run-art-pill")).not.toBeInTheDocument();
  const pane = screen.getByTestId("first-run-art-pane");
  expect(pane).toHaveAttribute("data-art-mode", "globe");
  expect(pane).toHaveAttribute("data-art-background", "#b09532");
  expect(pane).toHaveAttribute("data-art-arrow-color", "#eeede9");
  expect(pane).toHaveAttribute("data-art-count", "400");
  for (const label of ONBOARDING_LABELS) {
    expect(screen.queryByText(label)).not.toBeInTheDocument();
  }
}

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

  it("shows the gold arrow preview without onboarding labels", () => {
    render(
      <MemoryRouter>
        <OwnerSetup />
      </MemoryRouter>,
    );

    expectOwnerSetupPreview();
    expect(screen.queryByTestId("first-run-signed-in")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-log-out")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Create the owner account" })).toBeInTheDocument();
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
    expectOwnerSetupPreview();
    expect(screen.getByTestId("first-run-signed-in")).toHaveTextContent("ada@example.com");
    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
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
