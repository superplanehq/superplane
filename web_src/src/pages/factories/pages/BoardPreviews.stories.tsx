import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { githubAndSentryIntakeFactoriesFixture } from "../__fixtures__/backlogIntakeItemFixtures";
import { PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "../__fixtures__/factoryPageResponses";
import { LinesPage } from "./LinesPage";

const meta = {
  title: "Factories/Design previews/Board",
  component: LinesPage,
  parameters: {
    layout: "fullscreen",
    docs: { description: { component: "Design experiments and legacy board layouts. These differ from the app." } },
  },
} satisfies Meta<typeof LinesPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const IntakeBannersLegacy: Story = {
  name: "Intake banners (legacy)",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={githubAndSentryIntakeFactoriesFixture}
        previewFlags={{ addIntakeControl: true, columnAutomations: false }}
      />
    );
  },
};

export const AddIntakeControl: Story = {
  name: "Add intake control",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_FACTORY_LINES[0].id}?intake=1`}
      factoriesFixture={githubAndSentryIntakeFactoriesFixture}
      previewFlags={{ addIntakeControl: true }}
    />
  ),
};
