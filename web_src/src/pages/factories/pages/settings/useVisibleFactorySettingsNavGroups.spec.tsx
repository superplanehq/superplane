import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
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

describe("Codex navigation", () => {
  it("shows Codex without loading connection status", () => {
    const client = new QueryClient();
    render(
      <QueryClientProvider client={client}>
        <SettingsNavigation />
      </QueryClientProvider>,
    );
    expect(screen.getByRole("link", { name: "Codex" })).toHaveAttribute("href", "/account/codex-usage");
    expect(meListCodexUsage).not.toHaveBeenCalled();
  });
});
