import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoryLineEditPage } from "./FactoryLineEditPage";
import { FactoriesHarness } from "./__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID, REFUND_FACTORY_LINES } from "./__fixtures__/factoryPageResponses";

/**
 * Line editor page: name field + steps editor (app + trigger per step).
 * The step editor fetches per-app trigger nodes from the canvas API, which
 * returns an empty catch-all in Storybook — that's the intended empty-triggers
 * state you get before an app is scaffolded.
 */
const meta = {
  title: "Pages/Factories/LineEdit",
  component: FactoryLineEditPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof FactoryLineEditPage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Create flow: fresh line form with one empty step to fill in. */
export const Create: Story = {
  name: "Create",
  render: () => (
    <FactoriesHarness
      pathSuffix={`factories/${PRIMARY_FACTORY_ID}/lines/new`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

/** Edit flow: existing `plan-and-implement` line prefilled with 3 steps. */
export const EditExisting: Story = {
  name: "Edit Existing",
  render: () => {
    const existingLineId = REFUND_FACTORY_LINES[0]?.id ?? "";
    return (
      <FactoriesHarness
        pathSuffix={`factories/${PRIMARY_FACTORY_ID}/lines/${existingLineId}/edit`}
        factoriesFixture={defaultFactoriesFixture}
      />
    );
  },
};
