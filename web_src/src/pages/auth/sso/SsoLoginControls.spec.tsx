import { beforeEach, describe, expect, it, mock } from "bun:test";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { LAST_USED_LOGIN_METHOD_STORAGE_KEY } from "@/lib/lastUsedLoginMethod";

import { ContinueWithSso, SsoOrganizationPrompt } from "./SsoLoginControls";

describe("ContinueWithSso", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("shows the domain error when SSO is not available", async () => {
    globalThis.fetch = mock().mockResolvedValue(Response.json({ ssoAvailable: false }));
    const user = userEvent.setup();
    render(<ContinueWithSso redirect="" lastUsed={false} />);
    await user.click(screen.getByRole("button", { name: "Continue with SSO" }));
    await user.type(screen.getByLabelText("Work email"), "ada@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with SSO" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("SSO is not set up for this email domain.");
  });

  it("keeps the form when discovery fails", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 500 }));
    const user = userEvent.setup();
    render(<ContinueWithSso redirect="" lastUsed={false} />);
    await user.click(screen.getByRole("button", { name: "Continue with SSO" }));
    await user.type(screen.getByLabelText("Work email"), "ada@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with SSO" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Single Sign-On could not be checked. Try again.");
    expect(screen.getByLabelText("Work email")).toBeInTheDocument();
  });

  it("records SSO as the last used method before navigation", async () => {
    globalThis.fetch = mock().mockResolvedValue(
      Response.json({ ssoAvailable: true, startUrl: "/auth/sso/connection/start" }),
    );
    const user = userEvent.setup();
    render(<ContinueWithSso redirect="/acme" lastUsed />);
    await user.click(screen.getByRole("button", { name: "Continue with SSO" }));
    await user.type(screen.getByLabelText("Work email"), "ada@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with SSO" }));
    expect(window.localStorage.getItem(LAST_USED_LOGIN_METHOD_STORAGE_KEY)).toBe("sso");
  });
});

describe("SsoOrganizationPrompt", () => {
  it("keeps the other method when SSO is required", async () => {
    const onUseAnother = mock();
    const user = userEvent.setup();
    render(<SsoOrganizationPrompt required onContinue={() => undefined} onUseAnother={onUseAnother} />);
    expect(screen.getByText("Your organization requires SSO.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Use another method" }));
    expect(onUseAnother).toHaveBeenCalled();
  });

  it("keeps the other method when SSO is optional", async () => {
    const onUseAnother = mock();
    const user = userEvent.setup();
    render(<SsoOrganizationPrompt required={false} onContinue={() => undefined} onUseAnother={onUseAnother} />);
    await user.click(screen.getByRole("button", { name: "Use another method" }));
    expect(onUseAnother).toHaveBeenCalled();
  });
});
