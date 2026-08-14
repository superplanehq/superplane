import type { Meta, StoryObj } from "@storybook/react-vite";

import { defaultCanvasAppFixture, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_APPS,
} from "../__fixtures__/factoryPageResponses";
import { FactoryAppCanvasPage } from "./FactoryAppCanvasPage";

const plannerApp = REFUND_FACTORY_APPS[0];

function factoryOwnedCanvasFixture(): CanvasAppFixture {
  const baseCanvas = defaultCanvasAppFixture.canvas?.canvas as
    | { metadata?: Record<string, unknown>; spec?: unknown }
    | undefined;

  return {
    ...defaultCanvasAppFixture,
    organizationId: FACTORIES_ORGANIZATION_ID,
    canvasId: plannerApp.id ?? "app-refund-planner",
    canvas: {
      canvas: {
        ...baseCanvas,
        metadata: {
          ...(baseCanvas?.metadata ?? {}),
          id: plannerApp.id,
          name: plannerApp.name,
          description: plannerApp.description,
          factoryId: PRIMARY_FACTORY_ID,
        },
      },
    },
  };
}

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
      appFixture={factoryOwnedCanvasFixture()}
    />
  ),
};

export const ConfigureEditMode: Story = {
  name: "Configure edit mode",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerApp.id}?configure=1&from=automations`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={factoryOwnedCanvasFixture()}
    />
  ),
};

export const WithRun: Story = {
  name: "With run query",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerApp.id}?from=automations&run=${defaultCanvasAppFixture.publishedRunId ?? "run-1"}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={factoryOwnedCanvasFixture()}
    />
  ),
};
