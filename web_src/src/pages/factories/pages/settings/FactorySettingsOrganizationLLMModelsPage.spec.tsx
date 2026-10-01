import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoryOnboardingAgentHarness } from "@/api-client";
import { IntegrationsBasePathProvider } from "@/lib/integrationSettingsPaths";
import { TooltipProvider } from "@/ui/tooltip";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsOrganizationLLMModelsPage } from "./FactorySettingsOrganizationLLMModelsPage";

const saveModels = vi.fn();
const saveHostedModels = vi.fn();
const switchSource = vi.fn();
let canUpdate = true;
let hostedModels: Array<{ key: string; label: string; provider: string }> = [];
let selectedHostedModels: typeof hostedModels = [];

type BYOKQuery = {
  data: {
    connected?: boolean;
    integrationId?: string;
    selected?: Array<{ id: string; name: string }>;
    candidates?: Array<{ id: string; name: string }>;
  };
  isLoading: boolean;
  isError: boolean;
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
      isError: false,
      error: null,
    },
  useUpdateBYOKLLMModels: () => ({ mutateAsync: saveModels, isPending: false }),
  useOrganizationHostedLLMModels: () => ({
    data: { candidates: hostedModels, selected: selectedHostedModels },
    isLoading: false,
    isError: false,
  }),
  useUpdateOrganizationHostedLLMModels: () => ({ mutateAsync: saveHostedModels, isPending: false }),
  useSwitchFactoryModelSource: () => ({ mutateAsync: switchSource, isPending: false }),
}));

vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: () => ({ data: hostedModels, isLoading: false, isError: false }),
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
    isError: false,
    error: null,
  };
}

function setConnected(provider: string, candidateIds: string[], selectedIds = candidateIds) {
  byokByProvider[provider] = {
    data: {
      connected: true,
      integrationId: `int-${provider}`,
      selected: models(selectedIds),
      candidates: models(candidateIds),
    },
    isLoading: false,
    isError: false,
    error: null,
  };
}

function renderPage(agentHarness?: FactoryOnboardingAgentHarness) {
  const factory: FactoriesFactory = { ...REFUND_FACTORY, onboarding: { ...REFUND_FACTORY.onboarding, agentHarness } };
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
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

describe("FactorySettingsOrganizationLLMModelsPage", () => {
  beforeEach(() => {
    canUpdate = true;
    hostedModels = [
      { key: "hosted::anthropic::claude-sonnet-4-6", label: "anthropic/claude-sonnet-4-6", provider: "anthropic" },
    ];
    selectedHostedModels = [...hostedModels];
    saveHostedModels.mockReset();
    vi.mocked(showErrorToast).mockClear();
    vi.mocked(showSuccessToast).mockClear();
    saveHostedModels.mockImplementation(async (keys: string[]) => {
      selectedHostedModels = hostedModels.filter((model) => keys.includes(model.key));
      return { selected: selectedHostedModels };
    });
    saveModels.mockReset();
    saveModels.mockResolvedValue({});
    switchSource.mockReset();
    switchSource.mockResolvedValue({});
    setDisconnected("anthropic");
    setDisconnected("openai");
    setDisconnected("openrouter");
  });

  it("lists SuperPlane-hosted models when the workspace uses hosted models", () => {
    setConnected("anthropic", ["claude-opus-4-6"]);
    renderPage("AGENT_HARNESS_SUPERPLANE");

    expect(screen.getByTestId("llm-models-source-badge")).toHaveTextContent("SuperPlane");
    expect(screen.getByTestId("llm-models-hosted")).toHaveTextContent("anthropic/claude-sonnet-4-6");
    expect(screen.queryByText("claude-opus-4-6")).not.toBeInTheDocument();
  });

  it("keeps the organization hosted model list when the workspace uses its own key", () => {
    setConnected("anthropic", ["claude-opus-4-6"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    expect(screen.getByTestId("llm-models-source-badge")).toHaveTextContent("Your key");
    expect(screen.getByTestId("llm-models-hosted")).toHaveTextContent("anthropic/claude-sonnet-4-6");
    expect(screen.getByTestId("llm-models-provider-anthropic")).toBeInTheDocument();
  });

  it("shows only models from connected keys and links to the key integration", () => {
    setConnected("anthropic", ["claude-opus-4-6", "claude-sonnet-4-6"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    expect(screen.getByTestId("llm-models-source-badge")).toHaveTextContent("Your key");
    expect(screen.queryByTestId("llm-models-source")).not.toBeInTheDocument();
    expect(screen.queryByTestId("llm-models-provider-openai")).not.toBeInTheDocument();
    const key = screen.getByTestId("llm-models-key-anthropic");
    expect(key).toHaveTextContent("Your Claude key");
    expect(key).toHaveTextContent("Agents in this workspace run with this key.");
    expect(within(key).getByRole("link", { name: "Open Claude integration" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/settings/organization/integrations/int-anthropic",
    );
    expect(screen.getByText("2 of 2 models selected").className).not.toContain("text-red");
  });

  it("saves hosted selectable keys and disables Save when the selection is unchanged", async () => {
    const user = userEvent.setup();
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const checkbox = screen.getByRole("checkbox", { name: "anthropic/claude-sonnet-4-6" });
    const save = screen.getByRole("button", { name: "Save models" });
    expect(checkbox).toBeChecked();
    expect(save).toBeDisabled();
    expect(saveHostedModels).not.toHaveBeenCalled();
    await user.click(checkbox);
    expect(save).toBeEnabled();
    await user.click(checkbox);
    expect(save).toBeDisabled();
    await user.click(checkbox);
    await user.click(save);
    expect(saveHostedModels).toHaveBeenCalledWith([]);
    expect(screen.getByText("0 of 1 models selected")).toBeInTheDocument();
    expect(checkbox).not.toBeChecked();
  });

  it("keeps newly available hosted candidates unchecked and saves their keys", async () => {
    const user = userEvent.setup();
    hostedModels.push({ key: "hosted::openrouter::openai/gpt-5", label: "openai/gpt-5", provider: "openrouter" });
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const newModel = screen.getByRole("checkbox", { name: "openai/gpt-5" });
    expect(newModel).not.toBeChecked();
    await user.click(newModel);
    await user.click(screen.getByRole("button", { name: "Save models" }));
    expect(saveHostedModels).toHaveBeenCalledWith(hostedModels.map((model) => model.key));
  });

  it("shows candidates after an empty hosted save and distinguishes duplicate labels by provider", () => {
    selectedHostedModels = [];
    hostedModels.push({
      key: "hosted::openrouter::anthropic/claude-sonnet-4-6",
      label: "anthropic/claude-sonnet-4-6",
      provider: "openrouter",
    });
    renderPage("AGENT_HARNESS_SUPERPLANE");
    expect(screen.getByRole("checkbox", { name: "anthropic/claude-sonnet-4-6 (Anthropic)" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "anthropic/claude-sonnet-4-6 (OpenRouter)" })).not.toBeChecked();
    expect(screen.getByText("0 of 2 models selected")).toBeInTheDocument();
  });

  it("lets readers see hosted candidates without editing or saving", () => {
    canUpdate = false;
    renderPage("AGENT_HARNESS_SUPERPLANE");
    expect(screen.getByRole("checkbox", { name: "anthropic/claude-sonnet-4-6" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save models" })).toBeDisabled();
  });

  it("keeps the hosted draft when saving fails so the admin can retry", async () => {
    const user = userEvent.setup();
    saveHostedModels.mockRejectedValue(new Error("The installation model list changed."));
    renderPage("AGENT_HARNESS_SUPERPLANE");
    await user.click(screen.getByRole("checkbox", { name: "anthropic/claude-sonnet-4-6" }));
    await user.click(screen.getByRole("button", { name: "Save models" }));
    expect(showErrorToast).toHaveBeenCalledWith("The installation model list changed.");
    expect(showSuccessToast).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Save models" })).toBeEnabled();
    expect(screen.getByRole("checkbox", { name: "anthropic/claude-sonnet-4-6" })).not.toBeChecked();
  });

  it("shows the current key and offers the other providers", () => {
    setConnected("anthropic", ["claude-opus-4-6"]);
    setConnected("openrouter", ["openai/gpt-5"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    expect(screen.getByText("Your Claude key")).toBeInTheDocument();
    expect(screen.queryByText("Your OpenRouter key")).not.toBeInTheDocument();
    expect(screen.getByTestId("llm-models-connect-openrouter")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use SuperPlane" })).toBeInTheDocument();
  });

  it("asks for a provider when the workspace uses your keys without one", () => {
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    expect(
      within(screen.getByTestId("llm-models-empty-banner")).getByRole("link", { name: "Open Integrations" }),
    ).toHaveAttribute("href", "/org-1/workspaces/RF/settings/organization/integrations");
  });

  it("uses normal text colors when a key cannot list models", () => {
    byokByProvider.anthropic = {
      data: { connected: true, integrationId: "int-anthropic" },
      isLoading: false,
      isError: true,
      error: new Error("list failed"),
    };
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    const message = screen.getByText("Unable to list models from the connected key.");
    expect(message.className).toContain("text-muted-foreground");
    expect(message.className).not.toMatch(/text-destructive|text-red/);
    expect(screen.queryByText("0 of 0 models selected")).not.toBeInTheDocument();
  });

  it("shows a connected key when the catalog request fails before data loads", () => {
    byokByProvider.anthropic = {
      data: {},
      isLoading: false,
      isError: true,
      error: new Error("list failed"),
    };
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    expect(screen.getByTestId("llm-models-provider-anthropic")).toBeInTheDocument();
    expect(screen.getByText("Unable to list models from the connected key.")).toBeInTheDocument();
    expect(screen.queryByTestId("llm-models-empty-banner")).not.toBeInTheDocument();
  });

  it("saves the models that the user enables", async () => {
    const user = userEvent.setup();
    setConnected("openrouter", ["anthropic/claude-sonnet-4-6", "openai/gpt-5"], ["anthropic/claude-sonnet-4-6"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    const provider = screen.getByTestId("llm-models-provider-openrouter");
    const save = within(provider).getByRole("button", { name: "Save models" });
    expect(save).toBeDisabled();

    await user.click(within(provider).getByText("openai/gpt-5"));
    await user.click(save);

    expect(saveModels).toHaveBeenCalledWith({
      provider: "openrouter",
      allowedModels: ["anthropic/claude-sonnet-4-6", "openai/gpt-5"],
    });
  });

  it("disables changes when the user cannot update", () => {
    canUpdate = false;
    setConnected("openrouter", ["anthropic/claude-sonnet-4-6"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    expect(
      within(screen.getByTestId("llm-models-provider-openrouter")).getByRole("button", { name: "Save models" }),
    ).toBeDisabled();
    expect(within(screen.getByTestId("llm-models-hosted")).getByRole("button", { name: "Save models" })).toBeDisabled();
  });

  it("offers Connect on the hosted page and asks for a key only after confirm", async () => {
    const user = userEvent.setup();
    renderPage("AGENT_HARNESS_SUPERPLANE");

    expect(screen.getByTestId("llm-models-connect-anthropic")).toBeInTheDocument();
    await user.click(
      within(screen.getByTestId("llm-models-connect-anthropic")).getByRole("button", { name: "Connect" }),
    );

    expect(screen.getByRole("heading", { name: "Switch automations to Claude?" })).toBeInTheDocument();
    expect(screen.queryByTestId("llm-models-switch-api-key")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Switch to Claude" }));
    expect(screen.getByTestId("llm-models-switch-api-key")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save and switch" })).toBeDisabled();

    await user.type(screen.getByTestId("llm-models-switch-api-key"), "sk-test");
    await user.click(screen.getByRole("button", { name: "Save and switch" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "anthropic", apiKey: "sk-test" });
  });

  it("switches to a connected provider without asking for a key", async () => {
    const user = userEvent.setup();
    setConnected("anthropic", ["claude-opus-4-6"]);
    renderPage("AGENT_HARNESS_SUPERPLANE");

    await user.click(
      within(screen.getByTestId("llm-models-connect-anthropic")).getByRole("button", { name: "Connect" }),
    );
    await user.click(screen.getByRole("button", { name: "Switch to Claude" }));

    expect(screen.queryByTestId("llm-models-switch-api-key")).not.toBeInTheDocument();
    expect(switchSource).toHaveBeenCalledWith({ source: "anthropic", apiKey: undefined });
  });

  it("returns to SuperPlane from the provider page and still edits the model checklist", async () => {
    const user = userEvent.setup();
    setConnected("anthropic", ["claude-opus-4-6", "claude-sonnet-4-6"], ["claude-opus-4-6"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    const provider = screen.getByTestId("llm-models-provider-anthropic");
    expect(within(provider).getByRole("button", { name: "Save models" })).toBeDisabled();
    await user.click(within(provider).getByText("claude-sonnet-4-6"));
    await user.click(within(provider).getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenCalledWith({
      provider: "anthropic",
      allowedModels: ["claude-opus-4-6", "claude-sonnet-4-6"],
    });

    await user.click(screen.getByRole("button", { name: "Use SuperPlane" }));
    expect(screen.getByRole("heading", { name: "Switch automations to SuperPlane?" })).toBeInTheDocument();
    expect(screen.queryByTestId("llm-models-switch-api-key")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Switch to SuperPlane" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "hosted", apiKey: undefined });
  });
});
