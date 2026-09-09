import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AccountContextType } from "@/contexts/accountContextState";
import { factoryQueryKeys } from "@/hooks/useFactoryData";
import { meKeys } from "@/hooks/useMe";

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

function renderOnboarding(queryClient = new QueryClient()) {
  return render(
    <QueryClientProvider client={queryClient}>
      <OrganizationOnboardingRedirect
        renderWorkspace={(workspace, entryPath) => (
          <div data-entry-path={entryPath} data-testid="internal-workspace-key">
            {workspace.workspaceKey}
          </div>
        )}
      />
    </QueryClientProvider>,
  );
}

function renderOnboardingWithReresolve(queryClient = new QueryClient()) {
  let reresolve: (() => Promise<void>) | undefined;
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <OrganizationOnboardingRedirect
        renderWorkspace={(workspace, _entryPath, reresolveWorkspace) => {
          reresolve = reresolveWorkspace;
          return <div data-testid="internal-workspace-slug">{workspace.organizationSlug}</div>;
        }}
      />
    </QueryClientProvider>,
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

    await act(reresolve);

    await waitFor(() => expect(screen.getByTestId("internal-workspace-slug")).toHaveTextContent("dev-user-x1y2z3"));
    expect(location.replace).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("drops cached queries for the provisioned slug so a reused slug starts setup fresh", async () => {
    const queryClient = new QueryClient();
    // Cache left behind by an earlier onboarding organization that held the
    // "dev-user" slug before it was renamed.
    queryClient.setQueryData(["factories", "dev-user"], [{ id: "old-factory", onboarding: { step: "repo" } }]);
    queryClient.setQueryData(["integrations", "connected", "dev-user"], [{ metadata: { id: "old-integration" } }]);
    queryClient.setQueryData(["factories", "other-org"], [{ id: "other-factory" }]);

    renderOnboarding(queryClient);

    await screen.findByTestId("internal-workspace-key");
    expect(queryClient.getQueryData(["factories", "dev-user"])).toBeUndefined();
    expect(queryClient.getQueryData(["integrations", "connected", "dev-user"])).toBeUndefined();
    expect(queryClient.getQueryData(["factories", "other-org"])).toBeDefined();
  });

  it("keeps cached queries when a re-resolve returns the same slug", async () => {
    const queryClient = new QueryClient();
    const { reresolve } = renderOnboardingWithReresolve(queryClient);

    await screen.findByTestId("internal-workspace-slug");
    // Cache written by the wizard itself after the workspace was adopted.
    queryClient.setQueryData(["factories", "dev-user"], [{ id: "current-factory" }]);

    await act(reresolve);

    expect(queryClient.getQueryData(["factories", "dev-user"])).toBeDefined();
  });

  it("keeps seeded workspace queries when a re-resolve adopts a new slug", async () => {
    const queryClient = new QueryClient();
    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ organizationSlug: "dev-user", workspaceKey: "NEWWO" }),
    });
    vi.stubGlobal("fetch", fetchMock);
    const { reresolve } = renderOnboardingWithReresolve(queryClient);

    await screen.findByText("dev-user");

    const seededFactory = { id: "current-factory", key: "NEWWO" };
    const currentUser = { id: "account-1", permissions: [{ resource: "factories", action: "update" }] };
    queryClient.setQueryData(factoryQueryKeys.list("github-owner"), [seededFactory]);
    queryClient.setQueryData(factoryQueryKeys.detail("github-owner", seededFactory.id), seededFactory);
    queryClient.setQueryData(meKeys.me("dev-user"), currentUser);
    queryClient.setQueryData(meKeys.me("github-owner"), { id: "stale-user", permissions: [] });
    queryClient.setQueryData(["integrations", "connected", "github-owner"], [{ id: "stale-integration" }]);
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ organizationSlug: "github-owner", workspaceKey: "NEWWO" }),
    });

    await act(reresolve);

    await waitFor(() => expect(screen.getByTestId("internal-workspace-slug")).toHaveTextContent("github-owner"));
    expect(queryClient.getQueryData(factoryQueryKeys.list("github-owner"))).toEqual([seededFactory]);
    expect(queryClient.getQueryData(factoryQueryKeys.detail("github-owner", seededFactory.id))).toEqual(seededFactory);
    expect(queryClient.getQueryData(meKeys.me("github-owner"))).toEqual(currentUser);
    expect(queryClient.getQueryData(["integrations", "connected", "github-owner"])).toBeUndefined();
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
