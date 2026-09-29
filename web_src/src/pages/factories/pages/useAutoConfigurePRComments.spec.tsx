import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { useAutoConfigurePRComments } from "./useAutoConfigurePRComments";

const useIntegrationResources = vi.fn();
const createHandler = vi.fn();

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: (...args: unknown[]) => useIntegrationResources(...args),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: createHandler, isPending: false }),
}));

vi.mock("@/lib/errors", () => ({
  getApiErrorMessage: (error: unknown, fallback: string) => (error instanceof Error ? error.message : fallback),
}));

const baseProps = {
  githubIntegrationId: "gh-1",
  repository: "acme/app",
  onboardingComplete: true,
  canConfigure: true,
  handlersLoaded: true,
  hasDiscussionHandler: false,
  refetchHandlers: vi.fn(),
};

function catalog(bots: { id: string; name: string }[]) {
  return { data: bots, isError: false, isPending: false, isFetching: false };
}

describe("useAutoConfigurePRComments", () => {
  beforeEach(() => {
    createHandler.mockReset();
    createHandler.mockResolvedValue({});
    useIntegrationResources.mockReset();
    useIntegrationResources.mockReturnValue(catalog([{ id: "coderabbitai", name: "coderabbitai[bot]" }]));
  });

  it("auto-creates the discussion handler once onboarding is complete", async () => {
    const { result } = renderHook(() =>
      useAutoConfigurePRComments({ ...baseProps, organizationId: "org-a", factoryId: "factory-a" }),
    );

    expect(result.current.pending).toBe(true);
    await waitFor(() => expect(createHandler).toHaveBeenCalledTimes(1));
    expect(createHandler).toHaveBeenCalledWith(
      expect.objectContaining({
        source: "SOURCE_PULL_REQUEST_DISCUSSION",
        settings: expect.objectContaining({
          subject: { repository: "acme/app" },
          discussion: expect.objectContaining({
            mention: "@superplaneagent",
            ignoreBots: true,
            allowedBots: ["coderabbitai"],
          }),
        }),
      }),
    );
    await waitFor(() => expect(result.current.pending).toBe(false));
  });

  it("creates again when the workspace changes while the hook stays mounted", async () => {
    const { rerender } = renderHook(
      ({ organizationId, factoryId }: { organizationId: string; factoryId: string }) =>
        useAutoConfigurePRComments({ ...baseProps, organizationId, factoryId }),
      { initialProps: { organizationId: "org-a", factoryId: "factory-a" } },
    );

    await waitFor(() => expect(createHandler).toHaveBeenCalledTimes(1));
    expect(createHandler).toHaveBeenLastCalledWith(
      expect.objectContaining({
        settings: expect.objectContaining({
          discussion: expect.objectContaining({ allowedBots: ["coderabbitai"] }),
        }),
      }),
    );

    useIntegrationResources.mockReturnValue(catalog([{ id: "bugbot", name: "bugbot[bot]" }]));
    rerender({ organizationId: "org-b", factoryId: "factory-b" });

    await waitFor(() => expect(createHandler).toHaveBeenCalledTimes(2));
    expect(createHandler).toHaveBeenLastCalledWith(
      expect.objectContaining({
        settings: expect.objectContaining({
          discussion: expect.objectContaining({ allowedBots: ["bugbot"] }),
        }),
      }),
    );
  });

  it("keeps pending true until the current workspace's own request finishes", async () => {
    type Resolver = (value: object) => void;
    let resolveFirst: Resolver;
    let resolveSecond: Resolver;
    const firstDeferred = new Promise<object>((resolve) => {
      resolveFirst = resolve;
    });
    const secondDeferred = new Promise<object>((resolve) => {
      resolveSecond = resolve;
    });

    createHandler.mockReturnValue(firstDeferred);

    const { rerender, result } = renderHook(
      ({ organizationId, factoryId }: { organizationId: string; factoryId: string }) =>
        useAutoConfigurePRComments({ ...baseProps, organizationId, factoryId }),
      { initialProps: { organizationId: "org-a", factoryId: "factory-a" } },
    );

    await waitFor(() => expect(createHandler).toHaveBeenCalledTimes(1));

    createHandler.mockReturnValue(secondDeferred);
    rerender({ organizationId: "org-b", factoryId: "factory-b" });

    await waitFor(() => expect(createHandler).toHaveBeenCalledTimes(2));

    // Workspace A's request is the stale, earlier one. Resolving it must not
    // clear the pending state while workspace B's own request is in flight.
    resolveFirst!({});
    await waitFor(() => {
      expect(result.current.pending).toBe(true);
    });

    // Workspace B's own request finishing is what clears its pending state.
    resolveSecond!({});
    await waitFor(() => {
      expect(result.current.pending).toBe(false);
    });
  });

  it("clears pending state when switching to a workspace that already has a discussion handler", async () => {
    type Resolver = (value: object) => void;
    let resolveFirst: Resolver;
    const firstDeferred = new Promise<object>((resolve) => {
      resolveFirst = resolve;
    });
    createHandler.mockReturnValue(firstDeferred);

    const { rerender, result } = renderHook(
      (props: { organizationId: string; factoryId: string; hasDiscussionHandler: boolean }) =>
        useAutoConfigurePRComments({ ...baseProps, ...props }),
      { initialProps: { organizationId: "org-a", factoryId: "factory-a", hasDiscussionHandler: false } },
    );

    await waitFor(() => expect(createHandler).toHaveBeenCalledTimes(1));
    expect(result.current.pending).toBe(true);

    // Workspace B already has a discussion handler, so it needs no create of
    // its own. Pending must clear right away instead of waiting on A's stale
    // request, which never resolves in this scenario.
    rerender({ organizationId: "org-b", factoryId: "factory-b", hasDiscussionHandler: true });

    await waitFor(() => expect(result.current.pending).toBe(false));
    expect(createHandler).toHaveBeenCalledTimes(1);

    // Workspace A's request finishing afterward must not reintroduce pending.
    resolveFirst!({});
    await waitFor(() => expect(result.current.pending).toBe(false));
  });
});
