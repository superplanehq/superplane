import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

const meta = {
  title: "Factories/Pages/Settings/Model status",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const statusPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/model-status`;

export const ScoredAndEmptyCells: Story = {
  name: "Scored and empty cells",
  render: () => (
    <FactoriesHarness
      pathSuffix={statusPath}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        modelStatusCells: [
          { provider: "anthropic", model: "claude-sonnet-4-6", score: 100 },
          { provider: "openai", model: "gpt-5", score: 67 },
          { provider: "openrouter", model: "x-ai/grok-4.6", score: 0 },
          { provider: "anthropic", model: "claude-opus-4-6" },
        ],
      }}
    />
  ),
};

export const Loading: Story = {
  name: "Loading",
  render: () => (
    <FactoriesHarness
      pathSuffix={statusPath}
      factoriesFixture={{ ...defaultFactoriesFixture, modelStatusDelayMs: 30000 }}
    />
  ),
};

export const CouldNotLoad: Story = {
  name: "Could not load",
  render: () => (
    <FactoriesHarness
      pathSuffix={statusPath}
      factoriesFixture={{ ...defaultFactoriesFixture, modelStatusError: true }}
    />
  ),
};
