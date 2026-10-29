import { QueryClient, QueryClientProvider, useQueries } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { factoryOwnerEligibilityKey } from "./useFactoryOwnerEligibility";
import { factoryLineRunnerModelsQueryKey } from "./useFactoryLineRunnerModels";
import { selectableLLMModelsQueryKey } from "./useSelectableLLMModels";
import { factoryCodexConnectionKey } from "@/pages/factories/pages/onboarding/useFactoryCodexConnection";
import { meKeys } from "./useMe";

const { useWebSocketMock } = vi.hoisted(() => ({
  useWebSocketMock: vi.fn(),
}));

vi.mock("@/lib/reactUseWebsocket", () => ({
  useWebSocket: useWebSocketMock,
}));

import { useFactoryWebsocket } from "./useFactoryWebsocket";

afterEach(() => {
  vi.restoreAllMocks();
});

function lastCall() {
  const call = useWebSocketMock.mock.calls.at(-1);
  if (!call) throw new Error("useWebSocket was not invoked");
  return call;
}

async function emit(payload: unknown) {
  const [, options] = lastCall();
  const onMessage = options.onMessage as (event: MessageEvent<unknown>) => void;
  await act(async () => {
    onMessage(new MessageEvent("message", { data: JSON.stringify(payload) }));
  });
}

describe("useFactoryWebsocket subscription recovery", () => {
  it.each(["connection event", "reconnect"])("refreshes mounted subscription queries on %s", async (trigger) => {
    const queryClient = new QueryClient();
    const scopedKeys = (organizationId: string, factoryId: string) => [
      factoryOwnerEligibilityKey(organizationId, factoryId, "order-1", "member-1"),
      factoryLineRunnerModelsQueryKey(organizationId, factoryId, "hotfix", { orderId: "order-1" }),
      selectableLLMModelsQueryKey(organizationId, factoryId, "", { orderId: "order-1" }),
      factoryCodexConnectionKey(organizationId, factoryId, "member-1"),
    ];
    const currentKeys = [
      ...scopedKeys("org-1", "factory-1"),
      meKeys.me("org-1"),
      ["me", "org-1", "codex-usage", "member-1"],
    ];
    const unrelatedKeys = [
      ...scopedKeys("org-1", "other-factory"),
      ...scopedKeys("other-org", "factory-1"),
      meKeys.me("other-org"),
      ["me", "other-org", "codex-usage", "member-1"],
    ];
    const queryFns = [...currentKeys, ...unrelatedKeys].map(() => vi.fn().mockResolvedValue("updated"));
    const { unmount } = renderHook(
      () => {
        useFactoryWebsocket("org-1", "factory-1");
        return useQueries({
          queries: [...currentKeys, ...unrelatedKeys].map((queryKey, index) => ({
            queryKey,
            queryFn: queryFns[index],
            initialData: "cached",
            staleTime: Infinity,
          })),
        });
      },
      {
        wrapper: ({ children }: { children: ReactNode }) =>
          createElement(QueryClientProvider, { client: queryClient }, children),
      },
    );
    const onOpen = lastCall()[1].onOpen as () => void;
    act(() => onOpen());
    expect(queryFns.every((queryFn) => queryFn.mock.calls.length === 0)).toBe(true);

    if (trigger === "reconnect") {
      act(() => onOpen());
    } else {
      await emit({ event: "work_order_updated", payload: { factoryId: "other-factory", reason: "codex_connection" } });
      expect(queryFns.every((queryFn) => queryFn.mock.calls.length === 0)).toBe(true);
      await emit({ event: "work_order_updated", payload: { factoryId: "factory-1", reason: "codex_connection" } });
    }

    await waitFor(() => {
      for (const key of currentKeys) expect(queryClient.getQueryData(key)).toBe("updated");
    });
    for (const key of unrelatedKeys) expect(queryClient.getQueryData(key)).toBe("cached");
    unmount();
  });
});
