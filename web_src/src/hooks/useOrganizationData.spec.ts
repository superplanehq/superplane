import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { organizationsUpdateOrganization } = vi.hoisted(() => ({
  organizationsUpdateOrganization: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    organizationsUpdateOrganization,
  };
});

import { useUpdateOrganization } from "./useOrganizationData";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useUpdateOrganization", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    organizationsUpdateOrganization.mockResolvedValue({ data: {} });
    window.history.replaceState(null, "", "/onboarding");
  });

  it("sends the organization header outside organization routes", async () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useUpdateOrganization("org-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "GitHub Owner", slug: "github-owner" });
    });

    expect(organizationsUpdateOrganization).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: { "x-organization-id": "org-1" },
        path: { id: "org-1" },
      }),
    );
  });
});
