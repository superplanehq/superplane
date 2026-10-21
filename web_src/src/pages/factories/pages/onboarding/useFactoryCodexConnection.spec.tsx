import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { factoriesDescribeFactoryCodexConnection, factoriesStartFactoryCodexConnection } from "@/api-client";

import { useFactoryCodexConnection } from "./useFactoryCodexConnection";

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
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockReset();
    vi.mocked(factoriesStartFactoryCodexConnection).mockReset();
    vi.mocked(factoriesStartFactoryCodexConnection).mockResolvedValue({ data: {} } as never);
  });

  it("refreshes a ready legacy connection with no model catalog", async () => {
    vi.mocked(factoriesDescribeFactoryCodexConnection).mockResolvedValue({
      data: { connection: { status: "ready", availableModels: [] } },
    } as never);

    renderHook(() => useFactoryCodexConnection("org-1", "factory-1", true), { wrapper });

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
});
