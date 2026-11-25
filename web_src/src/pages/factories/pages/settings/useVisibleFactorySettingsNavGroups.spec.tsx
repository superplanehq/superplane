import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { meListCodexUsage } from "@/api-client";
import { useVisibleFactorySettingsNavGroups } from "./useVisibleFactorySettingsNavGroups";
import { buildFactorySettingsSearchIndex, searchFactorySettings } from "./settingsSearch";

let modelStatusEnabled = false;
let webhookFeatures: string[] = [];

vi.mock("@/api-client", () => ({ meListCodexUsage: vi.fn() }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ hasPermission: () => true, isLoading: false }),
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => {
      if (id === "model_status") return modelStatusEnabled;
      if (["factory-as-code", "factory_webhook_intake", "factory_custom_automations"].includes(id)) {
        return webhookFeatures.includes(id);
      }
      return true;
    },
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

describe("Workspace webhook discovery", () => {
  for (const features of [[], ["factory_custom_automations"], ["factory-as-code"], ["factory_webhook_intake"]]) {
    it(`gates webhook navigation with ${features.join(", ") || "no features"}`, () => {
      webhookFeatures = features;
      const view = render(
        <QueryClientProvider client={new QueryClient()}>
          <SettingsNavigation />
        </QueryClientProvider>,
      );
      const enabled = features.includes("factory-as-code") || features.includes("factory_webhook_intake");
      expect(screen.queryByRole("link", { name: "Webhooks" }) !== null).toBe(enabled);
      view.unmount();
      webhookFeatures = [];
    });
  }
});
