import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { readBitbucketInstallationAttempt, writeBitbucketInstallationAttempt } from "./bitbucketInstallationAttempt";
import { bitbucketInstallationAttemptMs } from "./bitbucketOnboardingPoll";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";
import { useFirstRunBitbucket } from "./useFirstRunBitbucket";

const onboarding = vi.hoisted(() => ({
  slugs: [] as string[],
  attemptActiveCalls: [] as boolean[],
}));

vi.mock("./useBitbucketOnboarding", () => ({
  useBitbucketOnboarding: (_organizationId: string, options: { attemptActive?: boolean }) => {
    onboarding.attemptActiveCalls.push(options.attemptActive === true);
    return {
      data: {
        installUrl: "https://bitbucket.example/install",
        installedWorkspaces: onboarding.slugs.map((slug) => ({ slug })),
      },
      isPending: false,
      isFetching: false,
      error: null,
      refetch: () => Promise.resolve(),
      startInstallation: { mutateAsync: () => Promise.resolve("https://bitbucket.example/install") },
      selectIdentity: { mutateAsync: () => Promise.resolve() },
    };
  },
}));

const blocking: FirstRunBlocking = {
  run: (_action: string, operation: () => Promise<void>) => operation(),
} as unknown as FirstRunBlocking;

function wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

function renderBitbucket(organizationId = "org-1") {
  return renderHook(({ organizationId }) => useFirstRunBitbucket({ organizationId, setupFinished: false, blocking }), {
    wrapper,
    initialProps: { organizationId },
  });
}

// Install flow: the user copies the link, then the page reloads (unmount and mount).
async function startInstallAndReload(organizationId = "org-1") {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: () => Promise.resolve() },
  });
  const first = renderBitbucket(organizationId);
  await act(async () => {
    await first.result.current.copyInstallLink();
  });
  expect(first.result.current.installationAttemptActive).toBe(true);
  first.unmount();
  return renderBitbucket(organizationId);
}

describe("useFirstRunBitbucket installation attempt", () => {
  beforeEach(() => {
    sessionStorage.clear();
    onboarding.slugs = ["acme"];
    onboarding.attemptActiveCalls = [];
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-29T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows no spinner without a stored attempt", () => {
    const { result } = renderBitbucket();

    expect(result.current.installationAttemptActive).toBe(false);
  });

  it("keeps the spinner and fast polling after a reload", async () => {
    const { result } = await startInstallAndReload();

    expect(result.current.installationAttemptActive).toBe(true);
    expect(onboarding.attemptActiveCalls.at(-1)).toBe(true);
  });

  it("stops the spinner and clears storage when a new workspace appears", async () => {
    const { result, rerender } = await startInstallAndReload();

    onboarding.slugs = ["acme", "new-team"];
    rerender({ organizationId: "org-1" });

    expect(result.current.installationAttemptActive).toBe(false);
    expect(readBitbucketInstallationAttempt("org-1")).toBeNull();
  });

  it("times out five minutes after the original click, not after the reload", async () => {
    const first = await startInstallAndReload();
    act(() => {
      vi.advanceTimersByTime(bitbucketInstallationAttemptMs - 60_000);
    });
    first.unmount();

    const { result } = renderBitbucket();
    expect(result.current.installationAttemptActive).toBe(true);

    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    expect(result.current.installationAttemptActive).toBe(false);
    expect(result.current.installationAttemptTimedOut).toBe(true);
    expect(readBitbucketInstallationAttempt("org-1")).toBeNull();
  });

  it("drops a stored attempt that expired before the reload", () => {
    writeBitbucketInstallationAttempt("org-1", {
      startedAt: Date.now() - bitbucketInstallationAttemptMs - 1,
      knownSlugs: ["acme"],
    });

    const { result } = renderBitbucket();

    expect(result.current.installationAttemptActive).toBe(false);
    expect(result.current.installationAttemptTimedOut).toBe(false);
    expect(readBitbucketInstallationAttempt("org-1")).toBeNull();
  });

  it("does not show the spinner of another organization", async () => {
    const { result, rerender } = await startInstallAndReload("org-1");

    rerender({ organizationId: "org-2" });

    expect(result.current.installationAttemptActive).toBe(false);
    expect(readBitbucketInstallationAttempt("org-1")).not.toBeNull();
  });
});
