import { QueryClient } from "@tanstack/react-query";
import type { NavigateFunction } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { factoryQueryKeys } from "@/hooks/useFactoryData";

import { advanceAfterGithubConnect } from "./useOnboardingPageModel";

describe("advanceAfterGithubConnect", () => {
  const factoryId = "factory-1";
  const factoryKey = "APP";
  const oldSlug = "old-org";
  const nextSlug = "new-org";

  let queryClient: QueryClient;
  let navigate: NavigateFunction;
  let locationReplace: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    queryClient = new QueryClient();
    navigate = vi.fn();
    locationReplace = vi.fn();
    vi.stubGlobal("location", { ...window.location, replace: locationReplace });
  });

  it("navigates to the repo step before re-resolving a new organization slug", async () => {
    const order: string[] = [];
    navigate = vi.fn(() => {
      order.push("navigate");
    }) as unknown as NavigateFunction;
    const reresolveWorkspace = vi.fn().mockImplementation(async () => {
      order.push("reresolve");
    });

    await advanceAfterGithubConnect({
      onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
      organizationId: oldSlug,
      nextSlug,
      factoryId,
      factoryKey,
      navigate,
      reresolveWorkspace,
      queryClient,
    });

    expect(order).toEqual(["navigate", "reresolve"]);
    expect(locationReplace).not.toHaveBeenCalled();
  });

  it("re-resolves the workspace and navigates client-side during initial onboarding", async () => {
    queryClient.setQueryData(factoryQueryKeys.list(oldSlug), [{ id: factoryId }]);
    queryClient.setQueryData(factoryQueryKeys.detail(oldSlug, factoryId), { id: factoryId, name: "Old" });
    const reresolveWorkspace = vi.fn().mockResolvedValue(undefined);

    await advanceAfterGithubConnect({
      onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
      organizationId: oldSlug,
      nextSlug,
      factoryId,
      factoryKey,
      navigate,
      reresolveWorkspace,
      queryClient,
    });

    expect(locationReplace).not.toHaveBeenCalled();
    expect(reresolveWorkspace).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith("/onboarding?attempt=attempt-1&step=repo", { replace: true });
  });

  it("seeds the factory caches for the new slug before re-resolving the workspace", async () => {
    queryClient.setQueryData(factoryQueryKeys.list(oldSlug), [{ id: factoryId }]);
    queryClient.setQueryData(factoryQueryKeys.detail(oldSlug, factoryId), { id: factoryId, name: "Old" });

    let seededListWhenResolving: unknown;
    let seededDetailWhenResolving: unknown;
    const reresolveWorkspace = vi.fn().mockImplementation(async () => {
      // The wizard re-renders with the new slug as soon as this resolves, so
      // the seeded caches must already be in place before it runs.
      seededListWhenResolving = queryClient.getQueryData(factoryQueryKeys.list(nextSlug));
      seededDetailWhenResolving = queryClient.getQueryData(factoryQueryKeys.detail(nextSlug, factoryId));
    });

    await advanceAfterGithubConnect({
      onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
      organizationId: oldSlug,
      nextSlug,
      factoryId,
      factoryKey,
      navigate,
      reresolveWorkspace,
      queryClient,
    });

    expect(seededListWhenResolving).toEqual([{ id: factoryId }]);
    expect(seededDetailWhenResolving).toEqual({ id: factoryId, name: "Old" });
  });

  it("navigates to the organization-scoped setup route outside initial onboarding", async () => {
    const reresolveWorkspace = vi.fn();

    await advanceAfterGithubConnect({
      onboardingEntryPath: null,
      organizationId: oldSlug,
      nextSlug,
      factoryId,
      factoryKey,
      navigate,
      reresolveWorkspace,
      queryClient,
    });

    expect(reresolveWorkspace).not.toHaveBeenCalled();
    expect(navigate).toHaveBeenCalledWith(`/${nextSlug}/workspaces/${factoryKey}/setup?step=repo`, { replace: true });
    expect(locationReplace).not.toHaveBeenCalled();
  });

  it("skips re-resolution and navigates client-side when the organization slug is unchanged", async () => {
    const reresolveWorkspace = vi.fn();

    await advanceAfterGithubConnect({
      onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
      organizationId: oldSlug,
      nextSlug: oldSlug,
      factoryId,
      factoryKey,
      navigate,
      reresolveWorkspace,
      queryClient,
    });

    expect(reresolveWorkspace).not.toHaveBeenCalled();
    expect(navigate).toHaveBeenCalledWith("/onboarding?attempt=attempt-1&step=repo", { replace: true });
    expect(locationReplace).not.toHaveBeenCalled();
  });

  it("navigates client-side when re-resolving the workspace fails", async () => {
    const reresolveWorkspace = vi.fn().mockRejectedValue(new Error("re-resolve failed"));

    await advanceAfterGithubConnect({
      onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
      organizationId: oldSlug,
      nextSlug,
      factoryId,
      factoryKey,
      navigate,
      reresolveWorkspace,
      queryClient,
    });

    expect(reresolveWorkspace).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith("/onboarding?attempt=attempt-1&step=repo", { replace: true });
    expect(locationReplace).not.toHaveBeenCalled();
  });

  it("navigates client-side when no re-resolution callback is available", async () => {
    await advanceAfterGithubConnect({
      onboardingEntryPath: "/onboarding?attempt=attempt-1&step=vcs&pick=newest",
      organizationId: oldSlug,
      nextSlug,
      factoryId,
      factoryKey,
      navigate,
      reresolveWorkspace: null,
      queryClient,
    });

    expect(navigate).toHaveBeenCalledWith("/onboarding?attempt=attempt-1&step=repo", { replace: true });
    expect(locationReplace).not.toHaveBeenCalled();
  });
});
