import type { Meta, StoryObj } from "@storybook/react-vite";

import { defaultCanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  factoryOwnedCanvasFixture,
  REFUND_IMPLEMENTER_APP,
  REFUND_PLANNER_APP,
  refundLineCanvasFixture,
} from "../__fixtures__/factoryOwnedCanvasFixture";
import {
  defaultFactoriesFixture,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { FactoryAppCanvasPage } from "./FactoryAppCanvasPage";

/**
 * Factory-embedded canvas Configure (edit). Run viewing moved to
 * Factories/Pages/Work Order Split Run.
 */
const meta = {
  title: "Factories/Pages/Factory App Canvas",
  component: FactoryAppCanvasPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactoryAppCanvasPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const plannerCanvasFixture = factoryOwnedCanvasFixture(REFUND_PLANNER_APP);
const plannerAppId = REFUND_PLANNER_APP.id ?? "app-refund-planner";
const implementerAppId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
const planAndImplementLineId = REFUND_FACTORY_LINES[0]?.id ?? "line-plan-and-implement";
const fromLinesPath = `workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?run=${LINE_RUN_IMPLEMENT_FAILED_ID}&from=lines&lineId=${planAndImplementLineId}`;

export const FromAutomations: Story = {
  name: "From Automations",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerAppId}?from=automations`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={plannerCanvasFixture}
    />
  ),
};

export const FromLines: Story = {
  name: "From Lines (deprecated)",
  tags: ["deprecated"],
  render: () => (
    <FactoriesHarness
      pathSuffix={fromLinesPath}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={refundLineCanvasFixture()}
    />
  ),
};

export const ConfigureEditMode: Story = {
  name: "Configure edit mode",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerAppId}?configure=1&agent=1&from=automations`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={plannerCanvasFixture}
    />
  ),
};

export const WithRun: Story = {
  name: "With run query (deprecated)",
  tags: ["deprecated"],
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${plannerAppId}?from=automations&run=${defaultCanvasAppFixture.publishedRunId ?? "run-1"}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={plannerCanvasFixture}
    />
  ),
};

export const EditWithLocalAgent: Story = {
  name: "Edit with a local agent",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?configure=1&agentPrompt=1&from=lines&lineId=${planAndImplementLineId}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={refundLineCanvasFixture()}
    />
  ),
};

export const ViewYaml: Story = {
  name: "View YAML",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?configure=1&yaml=1&from=lines&lineId=${planAndImplementLineId}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={refundLineCanvasFixture()}
    />
  ),
};

export const EditWorkspacePanels: Story = {
  name: "Edit workspace panels",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?configure=1&agent=1&blocks=1&from=lines&lineId=${planAndImplementLineId}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={refundLineCanvasFixture()}
    />
  ),
};
