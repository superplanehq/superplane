import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../../__fixtures__/setupStoryFixtures";
import { FactorySettingsAgentPage } from "./FactorySettingsAgentPage";

const meta = {
  title: "Factories/Pages/Settings/Primary repository",
  component: FactorySettingsAgentPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsAgentPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Connected: Story = {
  render: () => (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/agent`}
      factoriesFixture={factoriesFixtureWithSetupAnswers({ ...SETUP_ANSWERS.repository, defaultBranch: "main" })}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
    />
  ),
};
