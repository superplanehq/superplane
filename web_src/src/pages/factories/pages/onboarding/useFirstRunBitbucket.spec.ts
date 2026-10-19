import { renderHook, act } from "@testing-library/react";
import { createElement } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { bitbucketInstallationAttemptMs } from "./bitbucketOnboardingPoll";
import { useFirstRunBitbucket } from "./useFirstRunBitbucket";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";

const bitbucketOnboarding = vi.hoisted(() => ({
  data: undefined as Record<string, unknown> | undefined,
  refetch: vi.fn(),
}));

vi.mock("./useBitbucketOnboarding", () => ({
  useBitbucketOnboarding: () => ({
    data: bitbucketOnboarding.data,
    isPending: false,
    error: null,
    refetch: bitbucketOnboarding.refetch,
    startInstallation: { mutateAsync: vi.fn() },
    selectIdentity: { mutateAsync: vi.fn() },
  }),
}));

vi.mock("./useFirstRunGitHub", () => ({
  openGitHubWindow: () => undefined,
  navigateGitHubWindow: () => undefined,
}));

const blocking: FirstRunBlocking = {
  action: null,
  busy: false,
  begin: () => true,
  finish: () => undefined,
  setAction: () => undefined,
  run: async (_action, operation) => operation(),
  runUntilNavigation: async (_action, operation) => {
    await operation();
  },
};

function useHook(organizationId: string) {
  return useFirstRunBitbucket({ organizationId, setupFinished: false, blocking });
}

function renderHookInRouter(organizationId = "org-1") {
  return renderHook(() => useHook(organizationId), {
    wrapper: ({ children }: { children: React.ReactNode }) =>
      createElement(MemoryRouter, { initialEntries: [`/${organizationId}/setup`] }, children),
  });
}

function describeData(installedWorkspaces: Array<{ slug?: string }>): Record<string, unknown> {
  return { providerConfigured: true, identity: { login: "ada" }, installedWorkspaces, repositories: [] };
}

describe("useFirstRunBitbucket installation attempt", () => {
  beforeEach(() => {
    localStorage.clear();
    bitbucketOnboarding.data = describeData([]);
    bitbucketOnboarding.refetch = vi.fn();
  });

  it("keeps the installation attempt active after a remount", async () => {
    const first = renderHookInRouter();

    await act(async () => {
      await first.result.current.grantAccess();
    });
    expect(first.result.current.installationAttemptActive).toBe(true);

    const second = renderHookInRouter();

    expect(second.result.current.installationAttemptActive).toBe(true);
  });

  it("stops the spinner when the new workspace is confirmed", async () => {
    const first = renderHookInRouter();
    await act(async () => {
      await first.result.current.grantAccess();
    });

    bitbucketOnboarding.data = describeData([{ slug: "acme-team" }]);
    const second = renderHookInRouter();
    await act(async () => {});
    await act(async () => {});

    expect(second.result.current.installationAttemptActive).toBe(false);
  });

  it("ignores an attempt older than the five-minute window", () => {
    const key = "superplane:bitbucket-install-attempt:org-1";
    localStorage.setItem(
      key,
      JSON.stringify({ startedAt: Date.now() - bitbucketInstallationAttemptMs - 1, knownSlugs: [] }),
    );

    const { result } = renderHookInRouter();

    expect(result.current.installationAttemptActive).toBe(false);
    expect(result.current.installationAttemptTimedOut).toBe(false);
    expect(localStorage.getItem(key)).toBeNull();
  });
});
