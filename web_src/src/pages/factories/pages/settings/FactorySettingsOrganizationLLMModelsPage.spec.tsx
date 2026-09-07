import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { IntegrationsBasePathProvider } from "@/lib/integrationSettingsPaths";
import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsOrganizationLLMModelsPage } from "./FactorySettingsOrganizationLLMModelsPage";

const mutateAsync = vi.fn();
let canUpdate = true;

type BYOKQuery = {
  data: {
    connected?: boolean;
    selected?: Array<{ id: string; name: string }>;
    candidates?: Array<{ id: string; name: string }>;
  };
  isLoading: boolean;
  error: Error | null;
};

const byokByProvider: Record<string, BYOKQuery> = {};

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/hooks/useLLMModelAllowlists", () => ({
  BYOK_PROVIDERS: ["anthropic", "openai", "openrouter"],
  useBYOKLLMModels: (_organizationId: string, provider: string) =>
    byokByProvider[provider] ?? {
      data: { connected: false, selected: [], candidates: [] },
      isLoading: false,
      error: null,
    },
  useUpdateBYOKLLMModels: () => ({ mutateAsync, isPending: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdate, isLoading: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function models(ids: string[]) {
  return ids.map((id) => ({ id, name: id }));
}

function setDisconnected(provider: string) {
  byokByProvider[provider] = {
    data: { connected: false, selected: [], candidates: [] },
    isLoading: false,
    error: null,
  };
}

function setConnected(provider: string, candidateIds: string[], selectedIds = candidateIds) {
  byokByProvider[provider] = {
    data: {
      connected: true,
      selected: models(selectedIds),
      candidates: models(candidateIds),
    },
    isLoading: false,
    error: null,
  };
}

function renderPage() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <TooltipProvider>
          <IntegrationsBasePathProvider basePath="/org-1/workspaces/RF/settings/organization/integrations">
            <FactorySettingsLayoutContext.Provider
              value={{
                organizationId: "org-1",
                factoryId: REFUND_FACTORY.id ?? "factory-1",
                factory: REFUND_FACTORY,
              }}
            >
              <FactorySettingsOrganizationLLMModelsPage />
            </FactorySettingsLayoutContext.Provider>
          </IntegrationsBasePathProvider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsOrganizationLLMModelsPage", () => {
  beforeEach(() => {
    canUpdate = true;
    mutateAsync.mockReset();
    mutateAsync.mockResolvedValue({});
    setDisconnected("anthropic");
    setDisconnected("openai");
    setDisconnected("openrouter");
  });

  it("shows a landing banner when no provider is connected", () => {
    renderPage();

    expect(screen.getByTestId("llm-models-empty-banner")).toHaveTextContent("Connect a model provider");
    expect(
      within(screen.getByTestId("llm-models-empty-banner")).getByRole("link", { name: "Open Integrations" }),
    ).toHaveAttribute("href", "/org-1/workspaces/RF/settings/organization/integrations");
    expect(screen.getByTestId("factory-settings-llm-models-anthropic")).toHaveTextContent(
      "Connect Claude on Integrations, then select models.",
    );
    expect(screen.getByTestId("factory-settings-llm-models-openai")).toHaveTextContent(
      "Connect OpenAI on Integrations, then select models.",
    );
    expect(screen.getByTestId("factory-settings-llm-models-openrouter")).toHaveTextContent(
      "Connect OpenRouter on Integrations, then select models.",
    );
  });

  it("shows OpenRouter search and Claude and OpenAI connect copy when only OpenRouter is connected", () => {
    setConnected("openrouter", ["anthropic/claude-sonnet-4-6", "openai/gpt-5"]);

    renderPage();

    expect(screen.queryByTestId("llm-models-empty-banner")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Search OpenRouter models")).toBeInTheDocument();
    expect(screen.getByText("anthropic/claude-sonnet-4-6")).toBeInTheDocument();
    expect(screen.getByTestId("factory-settings-llm-models-anthropic")).toHaveTextContent(
      "Connect Claude on Integrations, then select models.",
    );
    expect(screen.getByTestId("factory-settings-llm-models-openai")).toHaveTextContent(
      "Connect OpenAI on Integrations, then select models.",
    );
  });

  it("filters OpenRouter candidates from search", async () => {
    const user = userEvent.setup();
    setConnected("openrouter", ["anthropic/claude-sonnet-4-6", "openai/gpt-5", "moonshotai/kimi-k2.6"]);

    renderPage();

    await user.type(screen.getByLabelText("Search OpenRouter models"), "kimi");
    const openrouter = screen.getByTestId("factory-settings-llm-models-openrouter");
    expect(within(openrouter).getByText("moonshotai/kimi-k2.6")).toBeInTheDocument();
    expect(within(openrouter).queryByText("anthropic/claude-sonnet-4-6")).not.toBeInTheDocument();
    expect(within(openrouter).queryByText("openai/gpt-5")).not.toBeInTheDocument();
  });

  it("saves the selected OpenRouter models", async () => {
    const user = userEvent.setup();
    setConnected("openrouter", ["anthropic/claude-sonnet-4-6", "openai/gpt-5"], ["anthropic/claude-sonnet-4-6"]);

    renderPage();

    const save = screen.getByRole("button", { name: "Save models" });
    expect(save).toBeDisabled();

    await user.click(screen.getByText("openai/gpt-5"));
    expect(save).toBeEnabled();
    await user.click(save);

    expect(mutateAsync).toHaveBeenCalledWith({
      provider: "openrouter",
      allowedModels: ["anthropic/claude-sonnet-4-6", "openai/gpt-5"],
    });
  });

  it("disables save when the user cannot update the organization", () => {
    canUpdate = false;
    setConnected("openrouter", ["anthropic/claude-sonnet-4-6"]);

    renderPage();

    expect(screen.getByRole("button", { name: "Save models" })).toBeDisabled();
  });
});
