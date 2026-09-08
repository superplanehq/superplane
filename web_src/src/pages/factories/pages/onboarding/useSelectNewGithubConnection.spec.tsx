import { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { NavigateFunction } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { OrganizationsIntegration } from "@/api-client";

import { advanceAfterGithubConnect } from "./advanceAfterGithubConnect";
import { useOnboardingGithubConnections } from "./useSelectNewGithubConnection";

const githubConnection: OrganizationsIntegration = {
  metadata: {
    id: "github-connection",
    name: "GitHub",
    integrationName: "github",
    createdAt: "2026-09-03T00:00:00Z",
  },
  status: {
    state: "ready",
    metadata: { owner: "forestileao" },
  },
};

describe("useOnboardingGithubConnections", () => {
  it("reports the newly connected GitHub integration to onboarding", async () => {
    const selectInstance = vi.fn();
    const onConnectionSelected = vi.fn();

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [{ name: "github", allInstances: [githubConnection], readyInstances: [githubConnection] }],
        openSection: "vcs",
        selectNewest: true,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    await waitFor(() => expect(onConnectionSelected).toHaveBeenCalledWith(githubConnection));
    expect(selectInstance).toHaveBeenCalledWith("github", "github-connection");
  });

  // Only the `pick=newest` round trip auto-selects. A ready connection from
  // an earlier pass, or one bound by an approved install request outside the
  // round trip, waits for the user to select the account.
  it("does not auto-select a ready connection without the URL hint", () => {
    const selectInstance = vi.fn();
    const onConnectionSelected = vi.fn();

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [{ name: "github", allInstances: [githubConnection], readyInstances: [githubConnection] }],
        openSection: "vcs",
        selectNewest: false,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    expect(onConnectionSelected).not.toHaveBeenCalled();
    expect(selectInstance).not.toHaveBeenCalled();
  });

  // The OAuth return also carries `pick=newest` while the new connect still
  // waits for the account choice. The wizard must show that picker, not jump
  // ahead with an older ready connection.
  it("does not auto-select an old ready connection while an account picker is pending", () => {
    const selectInstance = vi.fn();
    const onConnectionSelected = vi.fn();
    const pendingConnection: OrganizationsIntegration = {
      metadata: { id: "github-pending", name: "GitHub 2", integrationName: "github" },
      status: {
        state: "pending",
        metadata: { pendingInstallations: [{ id: "11", accountLogin: "forestileao" }] },
      },
    };

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [
          { name: "github", allInstances: [githubConnection, pendingConnection], readyInstances: [githubConnection] },
        ],
        openSection: "vcs",
        selectNewest: true,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    expect(onConnectionSelected).not.toHaveBeenCalled();
    expect(selectInstance).not.toHaveBeenCalled();
  });

  it("advances from vcs to repo on the same slug without re-resolving the workspace", async () => {
    const selectInstance = vi.fn();
    const navigate = vi.fn() as unknown as NavigateFunction;
    const reresolveWorkspace = vi.fn();
    const locationReplace = vi.fn();
    vi.stubGlobal("location", { ...window.location, replace: locationReplace });

    const onConnectionSelected = vi.fn(async () =>
      advanceAfterGithubConnect({
        onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
        organizationId: "dev-user",
        nextSlug: "dev-user",
        factoryId: "factory-1",
        factoryKey: "APP",
        navigate,
        reresolveWorkspace,
        queryClient: new QueryClient(),
      }),
    );

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [{ name: "github", allInstances: [githubConnection], readyInstances: [githubConnection] }],
        openSection: "vcs",
        selectNewest: true,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith("/onboarding?attempt=attempt-1&step=repo", { replace: true }),
    );
    expect(onConnectionSelected).toHaveBeenCalledTimes(1);
    expect(reresolveWorkspace).not.toHaveBeenCalled();
    expect(locationReplace).not.toHaveBeenCalled();
  });

  it("advances from vcs to repo exactly once without a full-page reload", async () => {
    const selectInstance = vi.fn();
    const navigate = vi.fn() as unknown as NavigateFunction;
    const reresolveWorkspace = vi.fn().mockResolvedValue(undefined);
    const locationReplace = vi.fn();
    vi.stubGlobal("location", { ...window.location, replace: locationReplace });

    const onConnectionSelected = vi.fn(async () =>
      advanceAfterGithubConnect({
        onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
        organizationId: "old-org",
        nextSlug: "new-org",
        factoryId: "factory-1",
        factoryKey: "APP",
        navigate,
        reresolveWorkspace,
        queryClient: new QueryClient(),
      }),
    );

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [{ name: "github", allInstances: [githubConnection], readyInstances: [githubConnection] }],
        openSection: "vcs",
        selectNewest: true,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith("/onboarding?attempt=attempt-1&step=repo", { replace: true }),
    );
    expect(onConnectionSelected).toHaveBeenCalledTimes(1);
    expect(reresolveWorkspace).toHaveBeenCalledTimes(1);
    expect(locationReplace).not.toHaveBeenCalled();
  });
});
