import { userEvent, within } from "storybook/test";
import type { Meta, StoryObj } from "@storybook/react-vite";

import type { FactoriesFactoryCodexConnection } from "@/api-client";
import { PermissionsContext } from "@/contexts/permissionsContextState";
import { usePermissions } from "@/contexts/usePermissions";
import { FEATURE_ORGANIZATION_BYOK } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";
import { FactorySettingsOrganizationLLMModelsPage } from "./FactorySettingsOrganizationLLMModelsPage";

const meta = {
  title: "Factories/Pages/Settings/Models",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;
export default meta;
type Story = StoryObj<typeof meta>;
const modelsPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/models`;
const features = [FEATURE_ORGANIZATION_BYOK];
const OPENROUTER_CANDIDATES = [
  "anthropic/claude-sonnet-4-6",
  "anthropic/claude-opus-4-6",
  "openai/gpt-5",
  "google/gemini-2.5-pro",
  "moonshotai/kimi-k2.6",
  "deepseek/deepseek-chat",
  "qwen/qwen3-coder",
  "meta-llama/llama-4-maverick",
];

function fixture(
  source: "hosted" | "byok" | "subscription",
  overrides: Partial<FactoriesFixture> = {},
): FactoriesFixture {
  return {
    ...defaultFactoriesFixture,
    factories: defaultFactoriesFixture.factories.map((factory) => ({
      ...factory,
      onboarding: {
        ...factory.onboarding,
        agentHarness:
          source === "hosted"
            ? "AGENT_HARNESS_SUPERPLANE"
            : source === "subscription"
              ? "AGENT_HARNESS_CODEX"
              : "AGENT_HARNESS_CLAUDE_CODE",
        codexSubscription: source === "subscription",
        agentIntegrationId: source === "byok" ? "int-openrouter" : undefined,
      },
    })),
    byokConnectedProviders: source === "byok" ? ["openrouter"] : [],
    byokCandidatesByProvider: { openrouter: OPENROUTER_CANDIDATES },
    codexConnection: {
      id: "personal-connection",
      status: "ready",
      eligible: true,
      accountLabel: "ada@example.com",
      availableModels: ["gpt-5.4", "gpt-5.3-codex"],
    },
    codexMembers: [
      {
        userId: "ada",
        displayName: "Ada Lovelace",
        email: "ada@example.com",
        observedAt: new Date().toISOString(),
        limits: [
          {
            limitId: "codex",
            limitName: "Codex",
            planType: "Plus",
            primary: {
              usedPercent: 25,
              windowDurationMins: "300",
              resetsAt: new Date(Date.now() + 3600000).toISOString(),
            },
            secondary: {
              usedPercent: 40,
              windowDurationMins: "10080",
              resetsAt: new Date(Date.now() + 86400000 * 4).toISOString(),
            },
          },
        ],
      },
      {
        userId: "grace",
        displayName: "Grace Hopper",
        email: "grace@example.com",
        observedAt: new Date().toISOString(),
        limits: [
          {
            limitId: "codex",
            limitName: "Codex",
            planType: "Pro",
            primary: {
              usedPercent: 10,
              windowDurationMins: "300",
              resetsAt: new Date(Date.now() + 7200000).toISOString(),
            },
            secondary: {
              usedPercent: 20,
              windowDurationMins: "10080",
              resetsAt: new Date(Date.now() + 86400000 * 6).toISOString(),
            },
          },
        ],
      },
    ],
    ...overrides,
  };
}
function policy(source: string, provider: string, ids: string[]) {
  return { [`${REFUND_FACTORY.id}:${source}:${provider}`]: { inheritParent: false, configuredModels: ids } };
}
const renderFixture = (state: FactoriesFixture) => (
  <FactoriesHarness pathSuffix={modelsPath} experimentalFeatures={features} factoriesFixture={state} />
);

export const NoProviders: Story = { render: () => renderFixture(fixture("byok", { byokConnectedProviders: [] })) };
export const HostedSelection: Story = {
  name: "SuperPlane workspace selection",
  render: () =>
    renderFixture(fixture("hosted", { workspaceModelPolicies: policy("hosted", "anthropic", ["claude-sonnet-4-6"]) })),
};
export const EmptyHostedSelection: Story = {
  name: "No SuperPlane models selected",
  render: () =>
    renderFixture(
      fixture("hosted", {
        workspaceModelPolicies: {
          ...policy("hosted", "anthropic", []),
          ...policy("hosted", "openai", []),
          ...policy("hosted", "openrouter", []),
        },
      }),
    ),
};
export const OpenRouterOnly: Story = {
  name: "OpenRouter connected",
  render: () =>
    renderFixture(
      fixture("byok", { workspaceModelPolicies: policy("byok", "openrouter", ["anthropic/claude-sonnet-4-6"]) }),
    ),
};
export const AllConnected: Story = {
  name: "Provider switching",
  render: () => renderFixture(fixture("byok", { byokConnectedProviders: ["anthropic", "openai", "openrouter"] })),
};
export const Subscription: Story = {
  render: () =>
    renderFixture(fixture("subscription", { workspaceModelPolicies: policy("subscription", "openai", ["gpt-5.4"]) })),
};
export const UnavailableSubscriptionModels: Story = {
  render: () =>
    renderFixture(
      fixture("subscription", {
        workspaceModelPolicies: policy("subscription", "openai", [
          "gpt-5.4",
          "gpt-native-model-with-a-long-identifier-unavailable-to-the-current-editor",
        ]),
      }),
    ),
};
export const NoConnectedMembers: Story = {
  render: () => renderFixture(fixture("subscription", { codexMembers: [] })),
};
export const MemberUsageUnavailable: Story = {
  render: () =>
    renderFixture(
      fixture("subscription", {
        codexMembers: [{ userId: "ada", displayName: "Ada Lovelace", email: "ada@example.com" }],
      }),
    ),
};
export const ConnectedMembersError: Story = {
  render: () => renderFixture(fixture("subscription", { codexMembersError: true })),
};
export const ConnectedMembersLoading: Story = {
  render: () => renderFixture(fixture("subscription", { codexMembersDelayMs: 30000 })),
};
export const CatalogError: Story = {
  render: () =>
    renderFixture(fixture("hosted", { workspaceModelPolicyError: "The model catalog is unavailable. Try again." })),
};
function ReadOnlyModels() {
  const permissions = usePermissions();
  return (
    <PermissionsContext.Provider
      value={{
        ...permissions,
        canAct: (resource, action) => action === "read" && permissions.canAct(resource, action),
      }}
    >
      <FactorySettingsOrganizationLLMModelsPage />
    </PermissionsContext.Provider>
  );
}
export const ReadOnly: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={modelsPath}
      experimentalFeatures={features}
      factoriesFixture={fixture("hosted")}
      pageOverrides={{ llmModels: ReadOnlyModels }}
    />
  ),
};
export const EmptySubscriptionCatalog: Story = {
  render: () =>
    renderFixture(
      fixture("subscription", {
        codexConnection: { eligible: true, status: "disconnected" } satisfies FactoriesFactoryCodexConnection,
      }),
    ),
};

export const Loading: Story = {
  render: () => renderFixture(fixture("hosted", { workspaceModelPolicyDelayMs: { GET: 30000 } })),
};
export const PendingSave: Story = {
  render: () =>
    renderFixture(
      fixture("byok", {
        workspaceModelPolicies: policy("byok", "openrouter", ["anthropic/claude-sonnet-4-6"]),
        workspaceModelPolicyDelayMs: { PUT: 8000 },
      }),
    ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole("button", { name: "Select all" }));
    await userEvent.click(canvas.getByRole("button", { name: "Save models" }));
  },
};
export const EmptySearch: Story = {
  render: () => renderFixture(fixture("byok")),
  play: async ({ canvasElement }) => {
    await userEvent.type(
      await within(canvasElement).findByRole("searchbox", { name: "Search OpenRouter models" }),
      "no-matching-model",
    );
  },
};
