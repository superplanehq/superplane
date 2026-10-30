import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  meCancelCodexConnection,
  factoriesListFactoryOwnerEligibility,
  factoriesListFactoryCodexUsage,
  meDescribeCodexConnection,
  meStartCodexConnection,
  type FactoriesFactoryCodexConnection,
} from "@/api-client";
import { showErrorToast } from "@/lib/toast";
import { unmockedSrc } from "@/test/unmockedModule";

import { renderPage } from "./organizationLLMModels.spec.render";
import { subscriptionModelSourceSwitchCases } from "./organizationLLMModels.subscription.spec.cases";

const saveModels = vi.fn();
const switchSource = vi.fn();
let canUpdate = true;
let canConnect = true;
let enabledFeatures = ["organization_byok", "codex_subscriptions"];
let codexConnection: FactoriesFactoryCodexConnection | undefined;
type Model = { id: string; name: string };
type Policy = { parent: Model[]; selected: Model[]; inheritParent: boolean; configuredModels: Model[] };
const policies: Record<string, Policy> = {};
const connectedProviders = new Set<string>();

function model(id: string, name = id): Model {
  return { id, name };
}
function policy(provider: string, source: string, ids: string[], selected = ids, inheritParent = false) {
  policies[`${source}:${provider}`] = {
    parent: ids.map((id) => model(id)),
    selected: ids.filter((id) => selected.includes(id)).map((id) => model(id)),
    configuredModels: inheritParent ? [] : selected.map((id) => model(id)),
    inheritParent,
  };
}

vi.mock("@/hooks/useOrganizationBilling", () => ({
  useOrganizationBilling: () => ({ data: { productFeatures: { byo_key: true } }, isSuccess: true, isPending: false }),
}));
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" },
    isSuccess: true,
    isPending: false,
    isError: false,
  }),
}));
vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: vi.fn() }));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => enabledFeatures.includes(id),
    organizationReady: true,
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useLLMModelAllowlists", () => ({
  BYOK_PROVIDERS: ["anthropic", "openai", "openrouter"],
  useBYOKLLMModels: (_organizationId: string, provider: string) => ({
    data: { connected: connectedProviders.has(provider), integrationId: `int-${provider}` },
    isLoading: false,
    isError: false,
    error: null,
  }),
  useFactoryLLMModels: (_organizationId: string, _factoryId: string, provider: string, source: string) => ({
    data: policies[`${source}:${provider}`] ?? { parent: [], selected: [], inheritParent: true, configuredModels: [] },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useUpdateFactoryLLMModels: () => ({ mutateAsync: saveModels, isPending: false }),
  useSwitchFactoryModelSource: () => ({ mutateAsync: switchSource, isPending: false }),
}));
vi.mock("@/api-client", () => ({
  ...unmockedSrc<Record<string, unknown>>("api-client/index.ts"),
  meDescribeCodexConnection: vi.fn(),
  meStartCodexConnection: vi.fn(),
  meCancelCodexConnection: vi.fn(),
  factoriesListFactoryOwnerEligibility: vi.fn(),
  factoriesListFactoryCodexUsage: vi.fn(),
}));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) =>
      (resource === "factories" && (action === "read" ? canConnect : canUpdate)) ||
      (resource === "integrations" && canUpdate),
    isLoading: false,
    currentUserId: "me",
  }),
}));
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [
      { metadata: { id: "me" }, spec: { displayName: "Current member" } },
      { metadata: { id: "other" }, spec: { displayName: "Other member" } },
    ],
    isLoading: false,
  }),
}));
vi.mock("@/lib/toast", () => ({ showSuccessToast: vi.fn(), showErrorToast: vi.fn() }));

describe("workspace LLM model settings", () => {
  beforeEach(() => {
    canUpdate = true;
    canConnect = true;
    enabledFeatures = ["organization_byok", "codex_subscriptions"];
    connectedProviders.clear();
    for (const key of Object.keys(policies)) delete policies[key];
    policy("anthropic", "hosted", ["claude-sonnet-4-6", "claude-opus-4-6"]);
    policy("openai", "byok", ["gpt-5", "gpt-4.1"]);
    codexConnection = { status: "disconnected", eligible: true };
    vi.mocked(meDescribeCodexConnection)
      .mockReset()
      .mockImplementation(async () => ({ data: { connection: codexConnection } }) as never);
    vi.mocked(meStartCodexConnection)
      .mockReset()
      .mockResolvedValue({ data: {} } as never);
    vi.mocked(meCancelCodexConnection)
      .mockReset()
      .mockResolvedValue({ data: {} } as never);
    vi.mocked(factoriesListFactoryOwnerEligibility)
      .mockReset()
      .mockImplementation(
        async () =>
          ({
            data: {
              subscription: true,
              ownershipChangeAllowed: true,
              members: [
                { userId: "me", eligible: codexConnection?.status === "ready" },
                { userId: "other", eligible: true },
              ],
            },
          }) as never,
      );
    vi.mocked(factoriesListFactoryCodexUsage)
      .mockReset()
      .mockResolvedValue({
        data: {
          members: [
            {
              userId: "other",
              displayName: "Other member",
              email: "other@example.com",
              observedAt: new Date().toISOString(),
              limits: [
                {
                  limitId: "codex",
                  primary: {
                    usedPercent: 25,
                    windowDurationMins: "300",
                    resetsAt: new Date(Date.now() + 3600000).toISOString(),
                  },
                },
              ],
            },
          ],
        },
      } as never);
    saveModels.mockReset().mockResolvedValue({});
    switchSource.mockReset().mockResolvedValue({});
    vi.mocked(showErrorToast).mockClear();
  });

  it("shows only the saved source and keeps it selected while a switch is canceled", async () => {
    connectedProviders.add("openai");
    renderPage("AGENT_HARNESS_SUPERPLANE");
    expect(screen.getByRole("tab", { name: "SuperPlane" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("searchbox", { name: "Search OpenAI models" })).not.toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("tab", { name: "Your API key" }));
    expect(screen.getByRole("tab", { name: "SuperPlane", hidden: true })).toHaveAttribute("aria-selected", "true");
    await userEvent.setup().click(screen.getByRole("button", { name: "Keep SuperPlane" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(switchSource).not.toHaveBeenCalled();
  });

  it("moves source focus with arrow keys and activates only with Enter", async () => {
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    screen.getByRole("tab", { name: "SuperPlane" }).focus();
    await user.keyboard("{ArrowRight}");
    expect(screen.getByRole("tab", { name: "Your API key" })).toHaveFocus();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await user.keyboard("{Enter}");
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("saves an explicit empty workspace selection without enabling inheritance", async () => {
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Deselect all" }));
    await user.click(screen.getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenCalledWith({
      provider: "anthropic",
      fundingSource: "hosted",
      allowedModels: [],
      inheritParent: false,
    });
    expect(screen.getByText("No models are enabled. Select models or use all available models.")).toBeInTheDocument();
  });

  it("saves inheritance separately for each hosted provider", async () => {
    policy("openai", "hosted", ["gpt-5"]);
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const anthropic = screen.getByRole("group", { name: "Anthropic models" });
    const user = userEvent.setup();
    await user.click(within(anthropic).getByRole("checkbox", { name: "Use all available models" }));
    await user.click(within(anthropic).getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenCalledWith({
      provider: "anthropic",
      fundingSource: "hosted",
      allowedModels: [],
      inheritParent: true,
    });
    expect(
      within(screen.getByRole("group", { name: "OpenAI models" })).getByRole("button", { name: "Save models" }),
    ).toBeDisabled();
  });

  it("edits only the active key with workspace update permission", async () => {
    connectedProviders.add("openai");
    renderPage("AGENT_HARNESS_CODEX");
    expect(screen.queryByRole("searchbox", { name: "Search Anthropic models" })).not.toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole("checkbox", { name: /gpt-4.1/i }));
    await user.click(screen.getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenCalledWith({
      provider: "openai",
      fundingSource: "byok",
      allowedModels: ["gpt-5"],
      inheritParent: false,
    });
  });

  it("preserves unavailable subscription IDs until they are removed", async () => {
    codexConnection = {
      status: "ready",
      eligible: true,
      accountLabel: "person@example.com",
      availableModels: ["gpt-5.4"],
    };
    policy("openai", "subscription", ["gpt-5.4"], ["gpt-5.4", "native-model-unavailable"]);
    renderPage("AGENT_HARNESS_CODEX", true);
    const unavailable = await screen.findByRole("checkbox", { name: /native-model-unavailable/ });
    expect(unavailable).toBeChecked();
    expect(screen.getByText("Unavailable from connected members")).toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole("checkbox", { name: /gpt-5.4/i }));
    expect(
      screen.getByText("No available models are enabled. Select an available model or use all available models."),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenLastCalledWith({
      provider: "openai",
      fundingSource: "subscription",
      allowedModels: ["native-model-unavailable"],
      inheritParent: false,
    });
    await user.click(unavailable);
    await user.click(screen.getByRole("button", { name: "Save models" }));
    expect(saveModels).toHaveBeenLastCalledWith({
      provider: "openai",
      fundingSource: "subscription",
      allowedModels: [],
      inheritParent: false,
    });
  });

  it("searches names and native IDs and reports no results", async () => {
    policies["hosted:anthropic"].parent[0].name = "Claude Sonnet 4.6";
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const search = screen.getByRole("searchbox", { name: "Search Anthropic models" });
    const user = userEvent.setup();
    await user.type(search, "claude-sonnet-4-6");
    expect(screen.getByRole("checkbox", { name: /Claude Sonnet 4.6/ })).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: /opus 4-6/ })).not.toBeInTheDocument();
    await user.clear(search);
    await user.type(search, "missing");
    expect(screen.getByText("No matching models. Try a different search.")).toBeInTheDocument();
  });

  it("retains the draft and shows an actionable error after a failed save", async () => {
    saveModels.mockRejectedValue(new Error("The model catalog changed. Reload the models and try again."));
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Deselect all" }));
    await user.click(screen.getByRole("button", { name: "Save models" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The model catalog changed.");
    expect(screen.getByRole("button", { name: "Save models" })).toBeEnabled();
  });

  it("lets readers inspect the active models but disables editing and source switching", () => {
    canUpdate = false;
    renderPage("AGENT_HARNESS_SUPERPLANE");
    expect(screen.getByRole("button", { name: "Save models" })).toBeDisabled();
    expect(screen.getByRole("tab", { name: "Your API key" })).toBeDisabled();
    expect(screen.getByRole("checkbox", { name: /sonnet 4-6/ })).toBeDisabled();
  });

  it("shows connected members and saved usage without personal connection controls", async () => {
    codexConnection = { status: "reconnect_required", eligible: true, accountLabel: "personal@example.com" };
    renderPage("AGENT_HARNESS_CODEX", true);
    expect(await screen.findByText("Other member")).toBeInTheDocument();
    expect(screen.getByText("other@example.com")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "5-hour window remaining" })).toHaveAttribute("aria-valuenow", "75");
    expect(screen.queryByText("personal@example.com")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /(?:Connect|Reconnect|Disconnect) ChatGPT|Refresh usage/ }),
    ).not.toBeInTheDocument();
    expect(meStartCodexConnection).not.toHaveBeenCalled();
  });

  it("shows an empty connected-member list without connection controls", async () => {
    vi.mocked(factoriesListFactoryCodexUsage).mockResolvedValue({ data: { members: [] } } as never);
    renderPage("AGENT_HARNESS_CODEX", true);
    expect(await screen.findByText("No members have a connected ChatGPT account.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect ChatGPT" })).not.toBeInTheDocument();
  });

  it("can retry a failed member usage request", async () => {
    vi.mocked(factoriesListFactoryCodexUsage).mockRejectedValueOnce(new Error("Unavailable"));
    renderPage("AGENT_HARNESS_CODEX", true);
    await userEvent.setup().click(await screen.findByRole("button", { name: "Try again" }));
    expect(await screen.findByText("Other member")).toBeInTheDocument();
  });

  subscriptionModelSourceSwitchCases({
    setConnection: (connection) => {
      codexConnection = connection;
    },
    switchSource,
  });
});
