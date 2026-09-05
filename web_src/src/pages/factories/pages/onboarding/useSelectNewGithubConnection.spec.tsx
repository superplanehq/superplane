import { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { NavigateFunction } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { OrganizationsIntegration } from "@/api-client";

import { advanceAfterGithubConnect } from "./useOnboardingPageModel";
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
        selectSingleInitial: false,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    await waitFor(() => expect(onConnectionSelected).toHaveBeenCalledWith(githubConnection));
    expect(selectInstance).toHaveBeenCalledWith("github", "github-connection");
  });

  it("selects the single ready connection on initial onboarding without the URL hint", async () => {
    const selectInstance = vi.fn();
    const onConnectionSelected = vi.fn();

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [{ name: "github", allInstances: [githubConnection], readyInstances: [githubConnection] }],
        openSection: "vcs",
        selectNewest: false,
        selectSingleInitial: true,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    await waitFor(() => expect(onConnectionSelected).toHaveBeenCalledWith(githubConnection));
    expect(selectInstance).toHaveBeenCalledWith("github", "github-connection");
  });

  it("does not auto-select without the URL hint outside initial onboarding", () => {
    const selectInstance = vi.fn();
    const onConnectionSelected = vi.fn();

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [{ name: "github", allInstances: [githubConnection], readyInstances: [githubConnection] }],
        openSection: "vcs",
        selectNewest: false,
        selectSingleInitial: false,
        selections: {},
        selectInstance,
        onConnectionSelected,
      }),
    );

    expect(onConnectionSelected).not.toHaveBeenCalled();
    expect(selectInstance).not.toHaveBeenCalled();
  });

  it("does not auto-select on initial onboarding when several connections are ready", () => {
    const selectInstance = vi.fn();
    const onConnectionSelected = vi.fn();
    const otherConnection: OrganizationsIntegration = {
      metadata: { id: "older-connection", integrationName: "github", createdAt: "2026-09-01T00:00:00Z" },
      status: { state: "ready", metadata: { owner: "acme" } },
    };

    renderHook(() =>
      useOnboardingGithubConnections({
        integrationData: [
          {
            name: "github",
            allInstances: [githubConnection, otherConnection],
            readyInstances: [githubConnection, otherConnection],
          },
        ],
        openSection: "vcs",
        selectNewest: false,
        selectSingleInitial: true,
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
        selectSingleInitial: false,
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
        selectSingleInitial: false,
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
