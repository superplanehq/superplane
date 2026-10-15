import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";

const { meCreateToken } = vi.hoisted(() => ({
  meCreateToken: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  meCreateToken,
}));

import { useCreateUserToken } from "./useUserTokens";

function createWrapper() {
  const queryClient = new QueryClient();
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useCreateUserToken", () => {
  beforeEach(() => {
    meCreateToken.mockReset();
    meCreateToken.mockResolvedValue({ data: { plaintext: "one-time-token" } });
    window.history.replaceState({}, "", "/admin/fleet-manager-setup");
  });

  it("sends the selected organization from a reserved admin route", async () => {
    const { result } = renderHook(() => useCreateUserToken("org-1"), { wrapper: createWrapper() });

    await act(async () => {
      await result.current.mutateAsync({ name: "Fleet Manager" });
    });

    expect(meCreateToken).toHaveBeenCalledWith(
      expect.objectContaining({
        body: { name: "Fleet Manager" },
        headers: { "x-organization-id": "org-1" },
      }),
    );
  });
});
