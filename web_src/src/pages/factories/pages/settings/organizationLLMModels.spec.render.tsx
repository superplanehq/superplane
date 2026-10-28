import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import type { FactoriesFactory, FactoryOnboardingAgentHarness } from "@/api-client";
import { IntegrationsBasePathProvider } from "@/lib/integrationSettingsPaths";
import { TooltipProvider } from "@/ui/tooltip";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsOrganizationLLMModelsPage } from "./FactorySettingsOrganizationLLMModelsPage";

export function renderPage(
  agentHarness?: FactoryOnboardingAgentHarness,
  codexSubscription = false,
  client?: QueryClient,
) {
  const factory: FactoriesFactory = {
    ...REFUND_FACTORY,
    onboarding: { ...REFUND_FACTORY.onboarding, agentHarness, codexSubscription },
  };
  return render(
    <QueryClientProvider client={client ?? new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <TooltipProvider>
          <IntegrationsBasePathProvider basePath="/org-1/workspaces/RF/settings/organization/integrations">
            <FactorySettingsLayoutContext.Provider
              value={{ organizationId: "org-1", factoryId: REFUND_FACTORY.id ?? "factory-1", factory }}
            >
              <FactorySettingsOrganizationLLMModelsPage />
            </FactorySettingsLayoutContext.Provider>
          </IntegrationsBasePathProvider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}
