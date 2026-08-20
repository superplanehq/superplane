import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useAccountOrganizations } from "./useAccountOrganizations";

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useAccountOrganizations", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", async () => {
      return new Response(
        JSON.stringify([{ id: "org-1", name: "SuperPlane" }, { id: 2, name: "skip-me" }, { name: "missing-id" }]),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps only organizations that have an id and a name", async () => {
    const { result } = renderHook(() => useAccountOrganizations(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data).toEqual([{ id: "org-1", name: "SuperPlane" }]);
  });
});
