import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

const meta = {
  title: "Factories/Pages/Settings/LLM Models",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const modelsPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/models`;

const OPENROUTER_CANDIDATES = [
  "anthropic/claude-sonnet-4-6",
  "anthropic/claude-opus-4-6",
  "anthropic/claude-haiku-4-5",
  "openai/gpt-5",
  "openai/gpt-4.1",
  "google/gemini-2.5-pro",
  "google/gemini-2.5-flash",
  "moonshotai/kimi-k2.6",
  "deepseek/deepseek-chat",
  "qwen/qwen3-coder",
  "meta-llama/llama-4-maverick",
  "mistralai/mistral-large",
];

export const NoProviders: Story = {
  name: "No providers",
  render: () => (
    <FactoriesHarness
      pathSuffix={modelsPath}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        byokConnectedProviders: [],
      }}
    />
  ),
};

export const OpenRouterOnly: Story = {
  name: "OpenRouter connected",
  render: () => (
    <FactoriesHarness
      pathSuffix={modelsPath}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        byokConnectedProviders: ["openrouter"],
        byokCandidatesByProvider: {
          openrouter: OPENROUTER_CANDIDATES,
        },
        byokSelectedByProvider: {
          openrouter: ["anthropic/claude-sonnet-4-6"],
        },
      }}
    />
  ),
};

export const AllConnected: Story = {
  name: "All providers connected",
  render: () => (
    <FactoriesHarness
      pathSuffix={modelsPath}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        byokCandidatesByProvider: {
          openrouter: OPENROUTER_CANDIDATES,
        },
      }}
    />
  ),
};
