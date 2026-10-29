import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  factoriesCancelFactoryCodexConnection,
  factoriesDescribeFactoryCodexConnection,
  factoriesStartFactoryCodexConnection,
} from "@/api-client";

import { useFactoryCodexConnection } from "./useFactoryCodexConnection";
import { factoryOwnerEligibilityKey } from "@/hooks/useFactoryOwnerEligibility";
import { factoryLineRunnerModelsQueryKey } from "@/hooks/useFactoryLineRunnerModels";

let canUpdate = true;
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => canUpdate }) }));

vi.mock("@/api-client", () => ({
  factoriesDescribeFactoryCodexConnection: vi.fn(),
  factoriesStartFactoryCodexConnection: vi.fn(),
  factoriesCancelFactoryCodexConnection: vi.fn(),
}));

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {children}
    </QueryClientProvider>
  );
}

describe("useFactoryCodexConnection", () => {
  beforeEach(() => {
    canUpdate = true;
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockReset();
    vi.mocked(factoriesStartFactoryCodexConnection).mockReset();
    vi.mocked(factoriesStartFactoryCodexConnection).mockResolvedValue({ data: {} } as never);
    vi.mocked(factoriesCancelFactoryCodexConnection).mockReset();
  });

  it.each([true, false])("refreshes a legacy catalog only with update permission: %s", async (allowed) => {
    canUpdate = allowed;
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockResolvedValue({
      data: { connection: { status: "ready", availableModels: [] } },
    } as never);

    const result = renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });
    await waitFor(() => expect(result.result.current.query.data?.status).toBe("ready"));
    if (!allowed) {
      expect(factoriesStartFactoryCodexConnection).not.toHaveBeenCalled();
      return;
    }

    await waitFor(() => expect(factoriesStartFactoryCodexConnection).toHaveBeenCalledTimes(1));
    expect(factoriesStartFactoryCodexConnection).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { factoryId: "factory-1" },
        body: { refreshCatalog: true },
      }),
    );
  });

  it("does not refresh a connection that already has models", async () => {
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockResolvedValue({
      data: { connection: { status: "ready", availableModels: ["gpt-5.6-sol"] } },
    } as never);

    const result = renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });
    await waitFor(() => expect(result.result.current.query.isSuccess).toBe(true));

    expect(factoriesStartFactoryCodexConnection).not.toHaveBeenCalled();
  });

  it("refreshes cached ownership and line models after sign-in completes", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    const eligibilityKey = factoryOwnerEligibilityKey("org-1", "factory-1");
    const catalogKey = factoryLineRunnerModelsQueryKey("org-1", "factory-1", "ship");
    client.setQueryData(eligibilityKey, { members: [{ userId: "member-1", eligible: false }] });
    client.setQueryData(catalogKey, { models: [] });
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockResolvedValue({
      data: { connection: { status: "ready", availableModels: ["gpt-5.6-sol"] } },
    } as never);
    const result = renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    });
    await waitFor(() => expect(result.result.current.query.isSuccess).toBe(true));
    await waitFor(() => expect(client.getQueryState(eligibilityKey)?.isInvalidated).toBe(true));
    expect(client.getQueryState(catalogKey)?.isInvalidated).toBe(true);
  });

  it("detects a saved sign-in after an initial status request fails", async () => {
    vi.mocked(factoriesDescribeFactoryCodexConnection)
      .mockRejectedValueOnce(new Error("temporarily unavailable"))
      .mockResolvedValue({ data: { connection: { status: "ready", availableModels: ["gpt-5.6-sol"] } } } as never);

    const result = renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });

    await waitFor(() => expect(result.result.current.query.data?.status).toBe("ready"), { timeout: 5000 });
    expect(factoriesStartFactoryCodexConnection).not.toHaveBeenCalled();
  });

  it("checks for a completed sign-in when the start response is lost", async () => {
    vi.mocked(factoriesDescribeFactoryCodexConnection)
      .mockResolvedValueOnce({ data: { connection: { status: "disconnected" } } } as never)
      .mockResolvedValue({ data: { connection: { status: "ready", availableModels: ["gpt-5.6-sol"] } } } as never);
    vi.mocked(factoriesStartFactoryCodexConnection).mockRejectedValue(new Error("response lost"));

    const result = renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });
    await waitFor(() => expect(result.result.current.query.data?.status).toBe("disconnected"));
    act(() => result.result.current.start.mutate());

    await waitFor(() => expect(result.result.current.query.data?.status).toBe("ready"));
    expect(factoriesStartFactoryCodexConnection).toHaveBeenCalledTimes(1);
  });

  it("shows the cancellation response while status refresh waits and polls until cleanup finishes", async () => {
    const pending = { data: { connection: { status: "pending", reservationHeld: true } } };
    const canceling = { data: { connection: { status: "canceling", reservationHeld: true } } };
    let finishStatusRefresh = (_response: typeof pending) => {};
    const statusRefresh = new Promise<typeof pending>((resolve) => {
      finishStatusRefresh = resolve;
    });
    vi.mocked(factoriesDescribeFactoryCodexConnection)
      .mockResolvedValueOnce(pending as never)
      .mockImplementationOnce(() => statusRefresh as never)
      .mockResolvedValue({ data: { connection: { status: "disconnected", reservationHeld: false } } } as never);
    vi.mocked(factoriesCancelFactoryCodexConnection).mockResolvedValue(canceling as never);

    const result = renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });
    try {
      await waitFor(() => expect(result.result.current.query.data?.status).toBe("pending"));
      act(() => result.result.current.cancel.mutate());

      await waitFor(() => expect(result.result.current.query.data?.status).toBe("canceling"));
      finishStatusRefresh(canceling);
      await waitFor(() => expect(result.result.current.query.data?.status).toBe("disconnected"), { timeout: 5000 });
    } finally {
      finishStatusRefresh(canceling);
      result.unmount();
    }
  });
});
