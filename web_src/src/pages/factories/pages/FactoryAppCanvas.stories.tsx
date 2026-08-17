import type { Meta, StoryObj } from "@storybook/react-vite";

import { defaultCanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { factoryOwnedCanvasFixture } from "../__fixtures__/factoryOwnedCanvasFixture";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_APPS,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { FactoryAppCanvasPage } from "./FactoryAppCanvasPage";

const plannerApp = REFUND_FACTORY_APPS[0];
const plannerCanvas = factoryOwnedCanvasFixture(plannerApp, PRIMARY_FACTORY_ID);
const plannerLine = REFUND_FACTORY_LINES[0];

/**
 * Factory-embedded canvas: view mode (read-only) or Configure (?configure=1) edit mode.
 */
const meta = {
  title: "Factories/Pages/Factory App Canvas",
  component: FactoryAppCanvasPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactoryAppCanvasPage>;

export default meta;

type Story = StoryObj<typeof meta>;

export const FromAutomations: Story = {
  name: "From Automations",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerApp.id}?from=automations`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={plannerCanvas}
    />
  ),
};

export const ConfigureEditMode: Story = {
  name: "Configure edit mode",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerApp.id}?configure=1&from=automations`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={plannerCanvas}
    />
  ),
};

export const FromLines: Story = {
  name: "From Lines configure",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerApp.id}?configure=1&from=lines&lineId=${plannerLine.id}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={plannerCanvas}
    />
  ),
};

export const WithRun: Story = {
  name: "With run query",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerApp.id}?from=automations&run=${defaultCanvasAppFixture.publishedRunId ?? "run-1"}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={plannerCanvas}
    />
  ),
};
