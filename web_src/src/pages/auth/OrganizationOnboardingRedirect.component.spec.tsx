import { render, screen, waitFor } from "@testing-library/react";
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

function renderOnboardingWithReresolve() {
  let reresolve: (() => Promise<void>) | undefined;
  const utils = render(
    <OrganizationOnboardingRedirect
      renderWorkspace={(workspace, _entryPath, reresolveWorkspace) => {
        reresolve = reresolveWorkspace;
        return <div data-testid="internal-workspace-slug">{workspace.organizationSlug}</div>;
      }}
    />,
  );
  return { ...utils, reresolve: () => reresolve!() };
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
    expect(fetch).toHaveBeenCalledWith(
      "/account/onboarding",
      expect.objectContaining({
        body: expect.stringContaining('"owner":"Dev User"'),
      }),
    );
  });

  it("shows an error when the account has no name", async () => {
    accountState.account.name = "";
    accountState.account.email = "";

    renderOnboarding();

    expect(await screen.findByText(/Could not start workspace setup/)).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("re-resolves the workspace under a new organization slug without reloading", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ organizationSlug: "dev-user", workspaceKey: "NEWWO" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const { reresolve } = renderOnboardingWithReresolve();

    const slug = await screen.findByTestId("internal-workspace-slug");
    expect(slug).toHaveTextContent("dev-user");

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ organizationSlug: "dev-user-x1y2z3", workspaceKey: "NEWWO" }),
    });

    await reresolve();

    await waitFor(() => expect(screen.getByTestId("internal-workspace-slug")).toHaveTextContent("dev-user-x1y2z3"));
    expect(location.replace).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("starts workspace setup without GitHub authorization", async () => {
    accountState.account.linked_accounts = [];
    accountState.account.providers = [];

    renderOnboarding();

    const workspace = await screen.findByTestId("internal-workspace-key");
    expect(workspace).toHaveTextContent("NEWWO");
    expect(location.replace).not.toHaveBeenCalled();
    expect(location.assign).not.toHaveBeenCalled();
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        "/account/onboarding",
        expect.objectContaining({
          body: expect.stringContaining('"owner":"Dev User"'),
        }),
      );
    });
  });
});
