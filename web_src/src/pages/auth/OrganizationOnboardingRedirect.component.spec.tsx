import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AccountContextType } from "@/contexts/accountContextState";

import { OrganizationOnboardingRedirect } from "./OrganizationOnboardingRedirect";

const accountState = vi.hoisted(() => ({
  account: {
    id: "account-1",
    name: "Dev User",
    email: "dev@superplane.local",
    avatar_url: "",
    installation_admin: false,
    has_password: true,
    linked_accounts: [{ provider: "github", username: "dev-user" }],
    providers: [],
  } as NonNullable<AccountContextType["account"]>,
}));

const location = vi.hoisted(() => ({
  replace: vi.fn(),
  assign: vi.fn(),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: accountState.account,
  }),
}));

function renderOnboarding() {
  return render(
    <OrganizationOnboardingRedirect
      renderWorkspace={(workspace, entryPath) => (
        <div data-entry-path={entryPath} data-testid="internal-workspace-key">
          {workspace.workspaceKey}
        </div>
      )}
    />,
  );
}

describe("OrganizationOnboardingRedirect", () => {
  beforeEach(() => {
    accountState.account = {
      id: "account-1",
      name: "Dev User",
      email: "dev@superplane.local",
      avatar_url: "",
      installation_admin: false,
      has_password: true,
      linked_accounts: [{ provider: "github", username: "dev-user" }],
      providers: [],
    };
    location.replace.mockReset();
    location.assign.mockReset();
    window.history.replaceState(null, "", "/onboarding");
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        pathname: "/onboarding",
        search: "",
        replace: location.replace,
        assign: location.assign,
      },
    });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ organizationSlug: "dev-user", workspaceKey: "NEWWO" }),
      }),
    );
  });

  it("keeps the provisional workspace behind the onboarding route", async () => {
    renderOnboarding();

    const workspace = await screen.findByTestId("internal-workspace-key");
    expect(workspace).toHaveTextContent("NEWWO");
    expect(workspace.dataset.entryPath).toMatch(/^\/onboarding\?attempt=[0-9a-f-]+$/);
    expect(location.replace).not.toHaveBeenCalled();
  });

  it("connects GitHub when the account has no GitHub identity", async () => {
    accountState.account.linked_accounts = [];
    accountState.account.providers = [];

    renderOnboarding();

    await waitFor(() => {
      expect(location.replace).toHaveBeenCalledWith("/auth/github?intent=connect&redirect=%2Fonboarding");
    });
    expect(fetch).not.toHaveBeenCalled();
  });

  it("shows a sign-in conflict and does not restart GitHub", async () => {
    accountState.account.linked_accounts = [];
    accountState.account.providers = [];
    window.history.replaceState(
      null,
      "",
      "/onboarding?auth_error=signin_method_in_use&provider=github&attempt=42d6ce14-6153-4390-85fe-3d15e9df53c9",
    );
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        pathname: "/onboarding",
        search: "?auth_error=signin_method_in_use&provider=github&attempt=42d6ce14-6153-4390-85fe-3d15e9df53c9",
        replace: location.replace,
        assign: location.assign,
      },
    });

    renderOnboarding();

    expect(
      await screen.findByText(
        "This GitHub identity already belongs to another SuperPlane account. Delete that account first.",
      ),
    ).toBeInTheDocument();
    expect(location.replace).not.toHaveBeenCalled();
    expect(fetch).not.toHaveBeenCalled();

    await userEvent.click(screen.getByRole("button", { name: "Connect GitHub" }));
    expect(location.assign).toHaveBeenCalledWith("/auth/github?intent=connect&redirect=%2Fonboarding");
  });

  it("shows a linked-account conflict and does not restart GitHub", async () => {
    accountState.account.linked_accounts = [];
    accountState.account.providers = [];
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        pathname: "/onboarding",
        search: "?auth_error=linked_account_in_use&provider=github&attempt=42d6ce14-6153-4390-85fe-3d15e9df53c9",
        replace: location.replace,
        assign: location.assign,
      },
    });

    renderOnboarding();

    expect(await screen.findByText("Another SuperPlane account already uses this GitHub account.")).toBeInTheDocument();
    expect(location.replace).not.toHaveBeenCalled();
    expect(fetch).not.toHaveBeenCalled();
  });
});
