import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { meListCodexUsage } from "@/api-client";
import { useVisibleFactorySettingsNavGroups } from "./useVisibleFactorySettingsNavGroups";
import { buildFactorySettingsSearchIndex, searchFactorySettings } from "./settingsSearch";

let modelStatusEnabled = false;

vi.mock("@/api-client", () => ({ meListCodexUsage: vi.fn() }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ hasPermission: () => true, isLoading: false }),
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => id !== "model_status" || modelStatusEnabled,
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useBillingProductEnabled", () => ({ useBillingProductEnabled: () => true }));
vi.mock("@/hooks/useOrganizationFleetManagement", () => ({
  useOrganizationFleetAccess: () => ({ showNavigation: true }),
}));

function SettingsNavigation({ query = "" }: { query?: string }) {
  const groups = useVisibleFactorySettingsNavGroups("org-1");
  const results = searchFactorySettings(buildFactorySettingsSearchIndex({ navGroups: groups }), query);
  return (
    <nav>
      {groups.flatMap((group) =>
        group.items.map((item) => (
          <a key={item.id} href={`/${item.scope}/${item.section}`}>
            {item.label}
          </a>
        )),
      )}
      {results.map((result) => (
        <span key={result.id} data-testid="search-result">
          {result.title}
        </span>
      ))}
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

describe("Model status discovery", () => {
  for (const enabled of [false, true]) {
    it(`${enabled ? "shows" : "hides"} navigation and keyword search when the feature is ${enabled ? "enabled" : "disabled"}`, () => {
      modelStatusEnabled = enabled;
      const view = render(
        <QueryClientProvider client={new QueryClient()}>
          <SettingsNavigation query="availability" />
        </QueryClientProvider>,
      );
      expect(screen.queryByRole("link", { name: "Model status" }) !== null).toBe(enabled);
      expect(screen.queryAllByTestId("search-result").some((result) => result.textContent === "Model status")).toBe(
        enabled,
      );
      view.unmount();
      modelStatusEnabled = false;
    });
  }
});
