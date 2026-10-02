import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { factoriesDescribeFactoryPullRequestMergeability } = vi.hoisted(() => ({
  factoriesDescribeFactoryPullRequestMergeability: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesDescribeFactoryPullRequestMergeability,
  factoriesMergeFactoryPullRequest: vi.fn(),
  factoriesRetryFactoryPullRequestWebhook: vi.fn(),
}));

import { factoryQueryKeys } from "./useFactoryData";
import {
  factoryPullRequestMergeabilityPollInterval,
  factoryPullRequestWebhookSetupPollMs,
  useFactoryPullRequestMergeability,
} from "./useFactoryPullRequestMerge";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function pollInterval(queryClient: QueryClient, pullRequestId: string) {
  const query = queryClient.getQueryCache().find({
    queryKey: factoryQueryKeys.pullRequestMergeability("org-1", "factory-1", pullRequestId),
  });
  return (
    query?.options as {
      refetchInterval?: (current: { state: { data?: { webhookSetupPending?: boolean } } }) => number | false;
    }
  ).refetchInterval;
}

describe("factoryPullRequestMergeabilityPollInterval", () => {
  it("keeps checking while webhook setup is pending", () => {
    expect(factoryPullRequestMergeabilityPollInterval({ webhookSetupPending: true })).toBe(
      factoryPullRequestWebhookSetupPollMs,
    );
  });

  it("stops checking when setup is not pending", () => {
    expect(factoryPullRequestMergeabilityPollInterval({ webhookSetupPending: false })).toBe(false);
    expect(factoryPullRequestMergeabilityPollInterval(undefined)).toBe(false);
  });
});

describe("useFactoryPullRequestMergeability", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("checks again while webhook setup is pending", async () => {
    factoriesDescribeFactoryPullRequestMergeability.mockResolvedValue({
      data: { mergeability: { canMerge: true, webhookSetupPending: true } },
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useFactoryPullRequestMergeability("org-1", "factory-1", "pr-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const interval = pollInterval(queryClient, "pr-1");
    expect(typeof interval).toBe("function");
    expect(interval?.({ state: { data: { webhookSetupPending: true } } })).toBe(factoryPullRequestWebhookSetupPollMs);
    expect(interval?.({ state: { data: { webhookSetupPending: false } } })).toBe(false);
  });
});
