import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { describe, expect, it, vi } from "bun:test";
import { factoryQueryKeys } from "./useFactoryData";
import { useImportFactoryPullRequest, useSearchFactoryPullRequestImports } from "./useFactoryPullRequestImports";
import { CodexActionContext, isCodexActionCancelled } from "@/pages/factories/codex/codexActionContext";
import type { WorkOrdersPage } from "@/pages/factories/lib/workOrderListPagination";

const importPR = vi.fn();
const searchPRs = vi.fn();
vi.mock("@/api-client", () => ({
  factoriesImportFactoryPullRequest: (...args: unknown[]) => importPR(...args),
  factoriesSearchFactoryPullRequestImportItems: (...args: unknown[]) => searchPRs(...args),
}));

function setup(connect = vi.fn(async () => true)) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return createElement(
      QueryClientProvider,
      { client },
      createElement(CodexActionContext.Provider, { value: connect }, children),
    );
  }
  return { client, ...renderHook(() => useImportFactoryPullRequest("org", "workspace"), { wrapper: Wrapper }) };
}

const input = { lineId: "selected-line", pullRequestUrl: "https://github.com/acme/app/pull/42" };
const connectionRequired = { details: [{ domain: "superplane.com", reason: "CODEX_CONNECTION_REQUIRED" }] };

describe("Pull request import mutation", () => {
  it("updates the board and detail caches and refreshes the candidate list", async () => {
    importPR.mockReset();
    const order = {
      id: "task",
      title: "Imported PR",
      state: "STATE_OPEN" as const,
      number: "42",
      pullRequests: [{ id: "pr", number: "42" }],
    };
    importPR.mockResolvedValue({ data: { order, created: true } });
    const { client, result } = setup();
    const pageKey = factoryQueryKeys.workOrdersPage("org", "workspace", ["STATE_OPEN"]);
    client.setQueryData(pageKey, { pages: [{ orders: [], hasNextPage: false }], pageParams: [undefined] });
    const candidatesKey = ["factories", "org", "workspace", "pull-request-imports", "", 5];
    client.setQueryData(candidatesKey, []);
    await act(async () => {
      await result.current.mutateAsync(input);
    });
    const board = client.getQueryData<InfiniteData<WorkOrdersPage>>(pageKey);
    expect(board?.pages[0].orders[0]).toMatchObject({
      id: "task",
      state: "STATE_OPEN",
      pullRequests: [{ id: "pr", number: "42" }],
    });
    expect(client.getQueryData(factoryQueryKeys.workOrderDetail("org", "workspace", "task"))).toEqual(order);
    expect(client.getQueryState(pageKey)?.isInvalidated).toBe(true);
    expect(client.getQueryState(candidatesKey)?.isInvalidated).toBe(true);
  });

  it("retries only after ChatGPT connection succeeds", async () => {
    importPR.mockReset();
    importPR.mockRejectedValueOnce(connectionRequired).mockResolvedValueOnce({ data: { order: { id: "task" } } });
    const connect = vi.fn(async () => true);
    const { result } = setup(connect);
    await act(async () => {
      await result.current.mutateAsync(input);
    });
    expect(connect).toHaveBeenCalledWith("org", "workspace");
    expect(importPR).toHaveBeenCalledTimes(2);
  });

  it("keeps the board unchanged when ChatGPT connection is canceled", async () => {
    importPR.mockReset();
    importPR.mockRejectedValueOnce(connectionRequired);
    const { client, result } = setup(vi.fn(async () => false));
    let canceled = false;
    await act(async () => {
      try {
        await result.current.mutateAsync(input);
      } catch (error) {
        canceled = isCodexActionCancelled(error);
      }
    });
    expect(canceled).toBe(true);
    expect(importPR).toHaveBeenCalledTimes(1);
    expect(client.getQueryData(factoryQueryKeys.workOrderDetail("org", "workspace", "task"))).toBeUndefined();
  });
});

describe("Pull request search cache", () => {
  it("clears previous candidates when the workspace changes", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const response = { data: { items: [{ title: "Workspace A PR", url: "https://github.com/acme/a/pull/42" }] } };
    let resolveNext: (value: typeof response) => void = () => {};
    searchPRs.mockReset();
    searchPRs.mockResolvedValueOnce(response).mockReturnValueOnce(
      new Promise<typeof response>((resolve) => {
        resolveNext = resolve;
      }),
    );
    function Wrapper({ children }: { children: ReactNode }) {
      return createElement(QueryClientProvider, { client }, children);
    }
    const { result, rerender } = renderHook(
      ({ factoryId }) => useSearchFactoryPullRequestImports("org", factoryId, "", 5, true),
      { initialProps: { factoryId: "workspace-a" }, wrapper: Wrapper },
    );
    await waitFor(() => expect(result.current.data?.[0].title).toBe("Workspace A PR"));
    rerender({ factoryId: "workspace-b" });
    expect(result.current.data).toBeUndefined();
    await act(async () => {
      resolveNext({ data: { items: [] } });
    });
    await waitFor(() => expect(result.current.data).toEqual([]));
  });
});
