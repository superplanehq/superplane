import { subscriptionModelSourceSwitchCases } from "./organizationLLMModels.subscription.spec.cases";
import { QueryClient } from "@tanstack/react-query";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  factoriesCancelFactoryCodexConnection,
  factoriesDescribeFactoryCodexConnection,
  factoriesStartFactoryCodexConnection,
  type FactoriesFactoryCodexConnection,
} from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { unmockedSrc } from "@/test/unmockedModule";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { renderPage } from "./organizationLLMModels.spec.render";
import { factoryCodexConnectionKey } from "../onboarding/useFactoryCodexConnection";

const saveModels = vi.fn();
const saveHostedModels = vi.fn();
const switchSource = vi.fn();
let canUpdate = true;
let enabledFeatures: string[] = ["organization_byok"];
let productAllowed = true;
let hostedModels: Array<{ key: string; label: string; provider: string }> = [];
let selectedHostedModels: typeof hostedModels = [];
let codexConnection: FactoriesFactoryCodexConnection | undefined;

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

vi.mock("@/hooks/useOrganizationBilling", () => ({
  useOrganizationBilling: () => ({
    data: { productFeatures: { byo_key: productAllowed } },
    isSuccess: true,
    isPending: false,
  }),
}));

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" },
    isSuccess: true,
    isPending: false,
    isError: false,
    refetch: async () => ({ data: { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" } }),
  }),
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (featureId: string) => enabledFeatures.includes(featureId),
    enabledExperimentalFeatures: enabledFeatures,
    isLoading: false,
    organizationReady: true,
  }),
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

vi.mock("@/api-client", () => ({
  ...unmockedSrc<Record<string, unknown>>("api-client/index.ts"),
  factoriesDescribeFactoryCodexConnection: vi.fn(),
  factoriesStartFactoryCodexConnection: vi.fn(),
  factoriesCancelFactoryCodexConnection: vi.fn(),
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

describe("FactorySettingsOrganizationLLMModelsPage", () => {
  beforeEach(() => {
    canUpdate = true;
    enabledFeatures = ["organization_byok", "codex_subscriptions"];
    productAllowed = true;
    hostedModels = [
      { key: "hosted::anthropic::claude-sonnet-4-6", label: "anthropic/claude-sonnet-4-6", provider: "anthropic" },
    ];
    selectedHostedModels = [...hostedModels];
    codexConnection = { status: "disconnected", eligible: true };
    vi.mocked(factoriesDescribeFactoryCodexConnection)
      .mockReset()
      .mockImplementation(async () => ({ data: { connection: codexConnection } }) as never);
    vi.mocked(factoriesStartFactoryCodexConnection)
      .mockReset()
      .mockResolvedValue({ data: {} } as never);
    vi.mocked(factoriesCancelFactoryCodexConnection)
      .mockReset()
      .mockResolvedValue({ data: {} } as never);
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

  it("starts ChatGPT reconnection from completed workspace settings", async () => {
    codexConnection = { status: "reconnect_required", eligible: true };
    renderPage("AGENT_HARNESS_CODEX", true);
    const connect = await screen.findByRole("button", { name: "Connect ChatGPT" });
    await waitFor(() => expect(connect).toBeEnabled());
    await userEvent.setup().click(connect);
    expect(factoriesStartFactoryCodexConnection).toHaveBeenCalledTimes(1);
    expect(switchSource).not.toHaveBeenCalled();
  });

  it("shows the ChatGPT device code and can cancel sign-in from settings", async () => {
    codexConnection = {
      status: "pending",
      eligible: true,
      verificationUrl: "https://auth.openai.com/codex/device",
      verificationCode: "ABCD-1234",
    };
    renderPage("AGENT_HARNESS_CODEX", true);
    expect(await screen.findByText("ABCD-1234")).toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: "Cancel sign-in" }));
    expect(factoriesCancelFactoryCodexConnection).toHaveBeenCalledTimes(1);
  });

  it("disables ChatGPT reconnection for readers", async () => {
    canUpdate = false;
    codexConnection = { status: "reconnect_required", eligible: true };
    renderPage("AGENT_HARNESS_CODEX", true);
    expect(await screen.findByRole("button", { name: "Connect ChatGPT" })).toBeDisabled();
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

  it("shows every model from the connected ChatGPT subscription", async () => {
    codexConnection = {
      status: "ready",
      availableModels: ["gpt-5.4", "gpt-5.3-codex"],
    };
    renderPage("AGENT_HARNESS_CODEX", true);

    expect(screen.getByTestId("llm-models-source-badge")).toHaveTextContent("ChatGPT");
    const subscription = await screen.findByTestId("llm-models-codex-subscription");
    expect(subscription).toHaveTextContent("All models available to this ChatGPT subscription are enabled.");
    expect(within(subscription).getByRole("checkbox", { name: "gpt-5.4" })).toBeChecked();
    expect(within(subscription).getByRole("checkbox", { name: "gpt-5.3-codex" })).toBeChecked();
    expect(screen.getByTestId("llm-models-change-source")).toBeInTheDocument();
  });

  it.each(["reconnect_required", "disconnected"])(
    "keeps the subscription source when ChatGPT is %s",
    async (status) => {
      setConnected("openai", ["gpt-5"]);
      codexConnection = { status, availableModels: ["gpt-5.4"] };
      renderPage("AGENT_HARNESS_CODEX", true);
      expect(
        await screen.findByText("Connect your Codex account again to load subscription models."),
      ).toBeInTheDocument();
      expect(screen.getByTestId("llm-models-source-badge")).toHaveTextContent("ChatGPT");
      expect(screen.queryByTestId("llm-models-provider-openai")).not.toBeInTheDocument();
      expect(screen.getByTestId("llm-models-change-source")).toBeInTheDocument();
    },
  );

  it("does not refresh a saved ChatGPT catalog when the workspace uses hosted models", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(factoryCodexConnectionKey("org-1", REFUND_FACTORY.id ?? "factory-1"), {
      status: "ready",
      availableModels: [],
    });
    await act(async () => {
      renderPage("AGENT_HARNESS_SUPERPLANE", false, client);
    });
    expect(screen.getByTestId("llm-models-source-badge")).toHaveTextContent("SuperPlane");
    expect(factoriesDescribeFactoryCodexConnection).toHaveBeenCalledTimes(1);
    expect(factoriesStartFactoryCodexConnection).not.toHaveBeenCalled();
  });

  subscriptionModelSourceSwitchCases({
    setConnection: (connection) => {
      codexConnection = connection;
    },
    switchSource,
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

  it("selects and deselects every SuperPlane model from the bulk toggle", async () => {
    const user = userEvent.setup();
    hostedModels.push({ key: "hosted::openrouter::openai/gpt-5", label: "openai/gpt-5", provider: "openrouter" });
    renderPage("AGENT_HARNESS_SUPERPLANE");

    const hosted = screen.getByTestId("llm-models-hosted");
    const bulkToggle = within(hosted).getByRole("button", { name: "Select all" });
    await user.click(bulkToggle);
    await user.click(within(hosted).getByRole("button", { name: "Save models" }));
    expect(saveHostedModels).toHaveBeenCalledWith(hostedModels.map((model) => model.key));

    await user.click(within(hosted).getByRole("button", { name: "Deselect all" }));
    await user.click(within(hosted).getByRole("button", { name: "Save models" }));
    expect(saveHostedModels).toHaveBeenLastCalledWith([]);
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

  it("selects all models from the bulk toggle", async () => {
    const user = userEvent.setup();
    setConnected("openrouter", ["model-a", "model-b"], ["model-a"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    const provider = screen.getByTestId("llm-models-provider-openrouter");
    const bulkToggle = within(provider).getByTestId("model-allowlist-bulk-toggle");
    expect(bulkToggle).toHaveTextContent("Select all");
    await user.click(bulkToggle);
    await user.click(within(provider).getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenCalledWith({ provider: "openrouter", allowedModels: ["model-a", "model-b"] });
  });

  it("deselects all models from the bulk toggle", async () => {
    const user = userEvent.setup();
    setConnected("openrouter", ["model-a", "model-b"], ["model-a", "model-b"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");

    const provider = screen.getByTestId("llm-models-provider-openrouter");
    await user.click(within(provider).getByTestId("model-allowlist-bulk-toggle"));
    await user.click(within(provider).getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenCalledWith({ provider: "openrouter", allowedModels: [] });
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

  it("hides provider controls when the product permission is denied", () => {
    productAllowed = false;
    renderPage("AGENT_HARNESS_SUPERPLANE");
    expect(screen.queryByTestId("llm-models-connect-anthropic")).not.toBeInTheDocument();
    expect(screen.queryByTestId("llm-models-connect-custom")).not.toBeInTheDocument();
  });

  it.each(["rollout", "permission"])("switches an existing key to hosted models without BYOK %s", async (gate) => {
    if (gate === "rollout") enabledFeatures = [];
    if (gate === "permission") productAllowed = false;
    setConnected("anthropic", ["claude-opus-4-6"]);
    renderPage("AGENT_HARNESS_CLAUDE_CODE");
    expect(screen.queryByTestId("llm-models-provider-anthropic")).not.toBeInTheDocument();
    expect(screen.queryByTestId("llm-models-empty-banner")).not.toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Use SuperPlane" }));
    await user.click(screen.getByRole("button", { name: "Switch to SuperPlane" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "hosted", apiKey: undefined });
  });

  it("hides the custom provider until both BYOK flags are on", () => {
    enabledFeatures = ["organization_byok"];
    renderPage("AGENT_HARNESS_SUPERPLANE");

    expect(screen.queryByTestId("llm-models-connect-custom")).not.toBeInTheDocument();
  });

  it("saves a custom provider URL, token, and API type", async () => {
    const user = userEvent.setup();
    enabledFeatures = ["organization_byok", "organization_byok_custom_provider"];
    renderPage("AGENT_HARNESS_SUPERPLANE");

    await user.click(within(screen.getByTestId("llm-models-connect-custom")).getByRole("button", { name: "Connect" }));
    await user.click(screen.getByRole("button", { name: "Switch to Custom provider" }));
    await user.type(screen.getByTestId("llm-models-custom-url"), "https://models.example/v1");
    await user.type(screen.getByTestId("llm-models-custom-token"), "secret-token");
    await user.click(screen.getByTestId("llm-models-custom-api-type"));
    await user.click(screen.getByRole("option", { name: "OpenAI-compatible" }));
    await user.click(screen.getByRole("button", { name: "Save and switch" }));

    expect(switchSource).toHaveBeenCalledWith({
      source: "custom",
      apiKey: "secret-token",
      baseUrl: "https://models.example/v1",
      apiType: "openai-compatible",
    });
    expect(screen.queryByText(/opencode go/i)).not.toBeInTheDocument();
  });
});
