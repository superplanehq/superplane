import { describe, expect, it } from "bun:test";
import { render, screen } from "@testing-library/react";

import { SsoSettingsView } from "./SsoSettingsView";

const domain = {
  id: "domain-1",
  domain: "example.com",
  txtName: "_superplane-verification.example.com",
  txtValue: "superplane-domain-verification=token",
  verifiedAt: "2026-11-14T00:00:00Z",
};

describe("SsoSettingsView", () => {
  it("shows the locked state when the license is absent and no connection exists", () => {
    render(<SsoSettingsView model={{ licensed: false, domains: [], tokens: [] }} />);
    expect(screen.getByTestId("sso-locked")).toHaveTextContent("This feature needs a SuperPlane Enterprise license.");
    expect(screen.getByText("Enterprise")).toBeInTheDocument();
  });

  it("shows a draft connection before a test", () => {
    render(
      <SsoSettingsView
        model={{
          licensed: true,
          domains: [],
          tokens: [],
          connection: { id: "conn", status: "STATUS_DRAFT", clientSecretSet: true },
        }}
      />,
    );
    expect(screen.getByText("Status: Draft")).toBeInTheDocument();
    expect(screen.getByTestId("sso-test-result")).toHaveTextContent("No test yet.");
    expect(screen.getByRole("button", { name: "Activate" })).toBeDisabled();
  });

  it("enables activation after a successful test and a verified domain", () => {
    render(
      <SsoSettingsView
        model={{
          licensed: true,
          domains: [domain],
          tokens: [],
          connection: { id: "conn", status: "STATUS_DRAFT", lastTest: { ok: true, email: "ada@example.com" } },
        }}
      />,
    );
    expect(screen.getByTestId("sso-test-result")).toHaveTextContent("Last test succeeded.");
    expect(screen.getByRole("button", { name: "Activate" })).toBeEnabled();
  });

  it("keeps disable and delete available when the license is absent", () => {
    render(
      <SsoSettingsView
        model={{
          licensed: false,
          domains: [domain],
          tokens: [{ id: "token-1", description: "Okta" }],
          connection: { id: "conn", status: "STATUS_ACTIVE", lastTest: { ok: true } },
        }}
      />,
    );
    expect(screen.getByText("Status: Active")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Disable" })).toBeEnabled();
    for (const button of screen.getAllByRole("button", { name: "Delete" })) {
      expect(button).toBeEnabled();
    }
    expect(screen.getByRole("button", { name: "Revoke" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Activate" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Create SCIM token" })).toBeDisabled();
  });

  it("shows an error", () => {
    render(<SsoSettingsView model={{ licensed: true, domains: [], tokens: [], error: "The sign-in test failed." }} />);
    expect(screen.getByRole("alert")).toHaveTextContent("The sign-in test failed.");
  });
});
