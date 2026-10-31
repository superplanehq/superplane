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
    const user = userEvent.setup();
    expect(screen.getByRole("status", { name: "SuperPlane is active" })).toBeInTheDocument();
    expect(screen.queryByRole("searchbox", { name: "Search OpenAI models" })).not.toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Change model source" })).toHaveLength(1);
    expect(screen.queryByRole("button", { name: "Select Your API key" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Change model source" }));
    expect(screen.getByRole("dialog", { name: "Change model source" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Current source" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Select Your API key" }));
    expect(screen.getByRole("status", { name: "SuperPlane is active", hidden: true })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Back to sources" }));
    expect(screen.getAllByRole("dialog", { hidden: true })).toHaveLength(1);
    expect(screen.getByRole("dialog", { name: "Change model source" })).toBeInTheDocument();
    expect(switchSource).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Select Your API key" }));
    await user.click(screen.getByRole("button", { name: "Keep SuperPlane" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Change model source" })).toHaveFocus();
    expect(switchSource).not.toHaveBeenCalled();
  });

  it("opens source choices and confirmation with the keyboard", async () => {
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    screen.getByRole("button", { name: "Change model source" }).focus();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await user.keyboard("{Enter}");
    expect(screen.getByRole("dialog", { name: "Change model source" })).toBeInTheDocument();
    screen.getByRole("button", { name: "Select Your API key" }).focus();
    await user.tab();
    expect(screen.getByRole("button", { name: "Select ChatGPT subscriptions" })).toHaveFocus();
    await user.keyboard("{Enter}");
    expect(screen.getByRole("button", { name: "Switch to Codex" })).toBeInTheDocument();
    expect(screen.getAllByRole("dialog", { hidden: true })).toHaveLength(1);
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

  it("connects another API key provider through the same source dialog", async () => {
    connectedProviders.add("openai");
    renderPage("AGENT_HARNESS_CODEX");
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Change model source" }));
    await user.click(screen.getByRole("button", { name: "Change provider" }));
    await user.click(screen.getByRole("combobox", { name: "New API key provider" }));
    await user.click(screen.getByRole("option", { name: "Claude" }));
    await user.click(screen.getByRole("button", { name: "Switch to Claude" }));
    expect(screen.getByRole("dialog", { name: "Your Claude key" })).toBeInTheDocument();
    expect(screen.getAllByRole("dialog", { hidden: true })).toHaveLength(1);
    expect(switchSource).not.toHaveBeenCalled();
    await user.type(screen.getByLabelText("Claude API key"), "provider-key");
    await user.click(screen.getByRole("button", { name: "Save and switch" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "anthropic", apiKey: "provider-key" });
    expect(await screen.findByRole("status", { name: "Your API key is active" })).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
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
    expect(screen.getByRole("button", { name: "Change model source" })).toBeDisabled();
    expect(screen.getByRole("checkbox", { name: /sonnet 4-6/ })).toBeDisabled();
  });

  it("shows connected members and saved usage without personal connection controls", async () => {
    codexConnection = { status: "reconnect_required", eligible: true, accountLabel: "personal@example.com" };
    renderPage("AGENT_HARNESS_CODEX", true);
    expect(await screen.findByText("Other member")).toBeInTheDocument();
    expect(screen.getByText("other@example.com")).toBeInTheDocument();
    expect(screen.getByText("5-hour window")).toBeInTheDocument();
    expect(screen.getByText("75% remaining")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Connected members" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Change default task owner" })).not.toBeInTheDocument();
    expect(screen.queryByText("personal@example.com")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /(?:Connect|Reconnect|Disconnect) ChatGPT|Refresh usage/ }),
    ).not.toBeInTheDocument();
    expect(meStartCodexConnection).not.toHaveBeenCalled();
  });

  it("shows an empty subscription workspace without owner controls for readers", async () => {
    canUpdate = false;
    vi.mocked(factoriesListFactoryCodexUsage).mockResolvedValue({ data: { members: [] } } as never);
    renderPage("AGENT_HARNESS_CODEX", true);
    expect(await screen.findByText("No members have a connected ChatGPT account.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Change default task owner" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Change model source" })).toBeDisabled();
  });

  it("shows unavailable usage for saved limits with no quota windows", async () => {
    vi.mocked(factoriesListFactoryCodexUsage).mockResolvedValue({
      data: { members: [{ userId: "other", displayName: "Other member", limits: [{ limitId: "codex" }] }] },
    } as never);
    renderPage("AGENT_HARNESS_CODEX", true);
    const row = await screen.findByRole("row", { name: /Other member/ });
    expect(within(row).getByText("Usage unavailable")).toBeInTheDocument();
  });

  it("labels each available window when a member has multiple quota windows", async () => {
    vi.mocked(factoriesListFactoryCodexUsage).mockResolvedValue({
      data: {
        members: [
          {
            userId: "other",
            displayName: "Other member",
            limits: [
              { limitId: "missing" },
              {
                limitId: "codex",
                primary: { usedPercent: 25, windowDurationMins: "300" },
                secondary: { usedPercent: 50, windowDurationMins: "10080" },
                credits: { hasCredits: true, balance: "4712" },
              },
              { limitId: "monthly", primary: { usedPercent: 10, windowDurationMins: "43200" } },
            ],
          },
        ],
      },
    } as never);
    renderPage("AGENT_HARNESS_CODEX", true);
    const row = await screen.findByRole("row", { name: /Other member/ });
    expect(within(row).getByText("5-hour window")).toBeInTheDocument();
    expect(within(row).getByText("75% remaining")).toBeInTheDocument();
    expect(within(row).getByText("7-day window")).toBeInTheDocument();
    expect(within(row).getByText("50% remaining")).toBeInTheDocument();
    expect(within(row).getByText("30-day window")).toBeInTheDocument();
    expect(within(row).getByText("90% remaining")).toBeInTheDocument();
    expect(within(row).getByText("Available credits")).toBeInTheDocument();
    expect(within(row).getByText("4,712 credits")).toBeInTheDocument();
    expect(within(row).queryByText("Usage unavailable")).not.toBeInTheDocument();
  });

  it.each([
    { name: "zero balance", credits: { hasCredits: false, balance: "0" }, expected: "0 credits" },
    { name: "unlimited balance", credits: { hasCredits: true, unlimited: true }, expected: "Unlimited" },
    { name: "unknown balance", credits: { hasCredits: true }, expected: "Available" },
    { name: "no credits", credits: { hasCredits: false }, expected: "0 credits" },
    { name: "fractional balance", credits: { hasCredits: true, balance: "4712.5" }, expected: "4,712.5 credits" },
    {
      name: "monthly allowance with an unknown balance",
      credits: { hasCredits: true, balance: null },
      individualLimit: { limit: "5000", used: "288.32", remainingPercent: 94 },
      expected: "4,712 credits",
      label: "Monthly credits",
    },
    {
      name: "monthly allowance without credits",
      individualLimit: { limit: "5000", used: "288.32", remainingPercent: 94 },
      expected: "4,712 credits",
      label: "Monthly credits",
    },
    {
      name: "exhausted monthly allowance",
      credits: { hasCredits: false },
      individualLimit: { limit: "5000", used: "5000.5", remainingPercent: 0 },
      expected: "0 credits",
      label: "Monthly credits",
    },
    {
      name: "balance before monthly allowance",
      credits: { hasCredits: true, balance: "12.5" },
      individualLimit: { limit: "5000", used: "288.32", remainingPercent: 94 },
      expected: "12.5 credits",
    },
    {
      name: "unlimited balance before monthly allowance",
      credits: { hasCredits: true, unlimited: true },
      individualLimit: { limit: "5000", used: "288.32", remainingPercent: 94 },
      expected: "Unlimited",
    },
  ])(
    "shows $name without quota windows",
    async ({ credits, individualLimit, expected, label = "Available credits" }) => {
      const resetsAt = new Date(Date.now() + 86400000 * 20).toISOString();
      vi.mocked(factoriesListFactoryCodexUsage).mockResolvedValue({
        data: {
          members: [
            {
              userId: "other",
              displayName: "Other member",
              limits: [
                {
                  limitId: "codex",
                  credits,
                  individualLimit: individualLimit ? { ...individualLimit, resetsAt } : undefined,
                },
              ],
            },
          ],
        },
      } as never);
      renderPage("AGENT_HARNESS_CODEX", true);
      const row = await screen.findByRole("row", { name: /Other member/ });
      expect(within(row).getByText(label)).toBeInTheDocument();
      expect(within(row).getByText(expected)).toBeInTheDocument();
      const reset = within(row).queryByText(`Resets: ${new Date(resetsAt).toLocaleString()}`);
      if (label === "Monthly credits") expect(reset).toBeInTheDocument();
      if (label !== "Monthly credits") expect(reset).not.toBeInTheDocument();
      expect(within(row).queryByText("Usage unavailable")).not.toBeInTheDocument();
      const user = userEvent.setup();
      await user.click(within(row).getByRole("button", { name: "Options for Other member" }));
      const details = screen.getByRole("menu");
      expect(within(details).getByText(expected)).toBeInTheDocument();
      expect(within(details).queryByText("Usage unavailable")).not.toBeInTheDocument();
    },
  );

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
