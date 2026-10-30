import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { meListCodexUsage } from "@/api-client";
import { useVisibleFactorySettingsNavGroups } from "./useVisibleFactorySettingsNavGroups";

vi.mock("@/api-client", () => ({ meListCodexUsage: vi.fn() }));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => true, isLoading: false }),
}));
vi.mock("@/hooks/useBillingProductEnabled", () => ({ useBillingProductEnabled: () => true }));

function SettingsNavigation() {
  const groups = useVisibleFactorySettingsNavGroups("org-1");
  return (
    <nav>
      {groups.flatMap((group) =>
        group.items.map((item) => (
          <a key={item.id} href={`/${item.scope}/${item.section}`}>
            {item.label}
          </a>
        )),
      )}
    </nav>
  );
}

describe("Codex usage navigation", () => {
  it.each([false, true])("shows Codex usage only when the user has a connection: %s", async (connected) => {
    const load = vi.mocked(meListCodexUsage);
    load.mockClear();
    load.mockResolvedValue({
      data: {
        connections: connected ? [{ id: "connection-1", factoryId: "another-workspace", status: "ready" }] : [],
      },
      error: undefined,
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = render(
      <QueryClientProvider client={queryClient}>
        <SettingsNavigation />
      </QueryClientProvider>,
    );
    try {
      await waitFor(() => expect(load).toHaveBeenCalledTimes(1));
      await waitFor(() => expect(queryClient.isFetching()).toBe(0));
      await waitFor(() => {
        const link = screen.queryByRole("link", { name: "Codex usage" });
        if (connected) {
          expect(link).toBeInTheDocument();
          return;
        }
        expect(link).not.toBeInTheDocument();
      });
    } finally {
      view.unmount();
      queryClient.clear();
    }
  });
});
