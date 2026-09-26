import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { OrganizationsIntegration } from "@/api-client";

import { integrationKeys } from "./useIntegrations";
import { useSyncGitHubConnection } from "./useSyncGitHubConnection";

const updateIntegration = vi.hoisted(() => vi.fn());
vi.mock("@/api-client/sdk.gen", () => ({
  organizationsUpdateIntegration: updateIntegration,
}));

function wrapper(queryClient: QueryClient) {
  return function TestQueryClientProvider({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function integration(id: string, owner: string): OrganizationsIntegration {
  return {
    metadata: { id, integrationName: "github" },
    status: { state: "pending", metadata: { owner } },
  };
}

describe("useSyncGitHubConnection", () => {
  beforeEach(() => {
    updateIntegration.mockReset();
  });

  it("replaces the synchronized connection without discarding other connections", async () => {
    const queryClient = new QueryClient();
    const stale = integration("github-1", "old-account");
    const other = integration("github-2", "other-account");
    const refreshed = integration("github-1", "new-account");
    queryClient.setQueryData(integrationKeys.connected("org-1"), [stale, other]);
    updateIntegration.mockResolvedValue({ data: { integration: refreshed } });
    const { result } = renderHook(() => useSyncGitHubConnection("org-1"), {
      wrapper: wrapper(queryClient),
    });

    let response: OrganizationsIntegration | undefined;
    await act(async () => {
      response = await result.current("github-1");
    });

    expect(response).toEqual(refreshed);
    expect(queryClient.getQueryData(integrationKeys.connected("org-1"))).toEqual([refreshed, other]);
    expect(updateIntegration).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { id: "org-1", integrationId: "github-1" },
        body: {},
      }),
    );
  });

  it("fails when the synchronization response has no integration", async () => {
    const queryClient = new QueryClient();
    updateIntegration.mockResolvedValue({ data: {} });
    const { result } = renderHook(() => useSyncGitHubConnection("org-1"), {
      wrapper: wrapper(queryClient),
    });

    await expect(result.current("github-1")).rejects.toThrow(
      "GitHub connection synchronization returned no integration",
    );
  });
});
