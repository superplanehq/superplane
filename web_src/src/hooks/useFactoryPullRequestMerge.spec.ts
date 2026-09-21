import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryPullRequest } from "@/api-client";

const { factoriesMergeFactoryPullRequest } = vi.hoisted(() => ({
  factoriesMergeFactoryPullRequest: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesMergeFactoryPullRequest,
  factoriesDescribeFactoryPullRequestMergeability: vi.fn(),
}));

vi.mock("./useFactoryWebsocket", () => ({
  invalidateFactoryWorkOrderQueries: vi.fn(),
}));

import { factoryPullRequestsKey } from "./useFactoryData";
import { useMergeFactoryPullRequest } from "./useFactoryPullRequestMerge";

const ORGANIZATION_ID = "org-1";
const FACTORY_ID = "factory-1";
const ORDER_ID = "wo-1";

const OPEN_PR: FactoriesFactoryPullRequest = {
  id: "pr-6812",
  provider: "PROVIDER_GITHUB",
  url: "https://github.com/acme/payments/pull/6812",
  number: "6812",
  state: "STATE_OPEN",
};

const MERGED_PR: FactoriesFactoryPullRequest = {
  ...OPEN_PR,
  state: "STATE_MERGED",
};

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useMergeFactoryPullRequest cache write", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("writes the merged pull request into the cached order list", async () => {
    factoriesMergeFactoryPullRequest.mockResolvedValue({ data: { pullRequest: MERGED_PR } });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    queryClient.setQueryData(factoryPullRequestsKey(ORGANIZATION_ID, FACTORY_ID, { workOrderIds: [ORDER_ID] }), [
      OPEN_PR,
    ]);

    const { result } = renderHook(() => useMergeFactoryPullRequest(ORGANIZATION_ID, FACTORY_ID, ORDER_ID), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        pullRequestId: OPEN_PR.id!,
        mergeMethod: "MERGE_METHOD_SQUASH",
        expectedHeadSha: "abc123",
      });
    });

    await waitFor(() => {
      const cached = queryClient.getQueryData<FactoriesFactoryPullRequest[]>(
        factoryPullRequestsKey(ORGANIZATION_ID, FACTORY_ID, { workOrderIds: [ORDER_ID] }),
      );
      expect(cached?.find((pullRequest) => pullRequest.id === OPEN_PR.id)?.state).toBe("STATE_MERGED");
    });
  });
});
