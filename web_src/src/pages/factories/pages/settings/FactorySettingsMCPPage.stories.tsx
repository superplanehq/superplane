import type { Meta, StoryObj } from "@storybook/react-vite";
import {
  FEATURE_SUPERPLANE_MCP_SERVER,
  FEATURE_WORKSPACE_AGENT,
  FEATURE_WORKSPACE_MCP,
  FEATURE_WORKSPACE_SKILLS,
} from "@/lib/experimentalFeatures";
import { cn } from "@/lib/utils";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  CONFIGURED_MCP_RESOURCES,
  CONFIGURED_MCP_RESOURCE_TOOLS_BY_ID,
  CONFIGURED_SKILLS,
} from "../../__fixtures__/agentResourceFixtures";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";
import { factorySettingsNarrowSectionBodyClassName } from "../factoryPageLayoutStyles";
import { FactorySettingsLayout } from "./FactorySettingsLayout";
import { CrossRepoContextCard } from "./WorkspaceCrossRepoSection";
import { WorkspaceDefaultModelSection } from "./WorkspaceDefaultModelSection";
import { WorkspaceSetupStepsSection } from "./WorkspaceSetupStepsSection";

const meta = {
  title: "Factories/Pages/Settings/Agent",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/agent`;

const agentExperimentalFeatures = [
  FEATURE_WORKSPACE_AGENT,
  FEATURE_WORKSPACE_MCP,
  FEATURE_WORKSPACE_SKILLS,
  FEATURE_SUPERPLANE_MCP_SERVER,
];

function agentHarness(fixture: FactoriesFixture) {
  return (
    <FactoriesHarness
      pathSuffix={agentPath}
      experimentalFeatures={agentExperimentalFeatures}
      factoriesFixture={fixture}
    />
  );
}

/** Agent page with no outbound MCP servers or skills. */
export const Empty: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
    }),
};

/** Outbound MCP servers and skills configured. */
export const Configured: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      agentResourceToolsById: CONFIGURED_MCP_RESOURCE_TOOLS_BY_ID,
      agentResourcesByFactoryId: {
        [PRIMARY_FACTORY_ID]: [...CONFIGURED_MCP_RESOURCES, ...CONFIGURED_SKILLS],
      },
    }),
};

/** Bash commands in the setup-steps field. The Agent page does not show this field. */
export const SetupSteps: Story = {
  render: () => (
    <div className="min-h-screen bg-sidebar dark:bg-background">
      <div className={cn(factorySettingsNarrowSectionBodyClassName, "pt-14")}>
        <WorkspaceSetupStepsSection canUpdate commands={"npm ci\ngo mod download"} />
      </div>
    </div>
  ),
};

/** Model choices for the Storybook preview. The Agent page does not show this field. */
const previewModels = [
  { value: "claude-sonnet", label: "Claude Sonnet" },
  { value: "claude-opus", label: "Claude Opus" },
  { value: "gpt-5", label: "GPT-5" },
];

/** Default model field. The Agent page does not show this field. */
export const DefaultModel: Story = {
  render: () => (
    <div className="min-h-screen bg-sidebar dark:bg-background">
      <div className={cn(factorySettingsNarrowSectionBodyClassName, "pt-14")}>
        <WorkspaceDefaultModelSection canUpdate models={previewModels} />
      </div>
    </div>
  ),
};

/** Extra repositories in the cross-repo list. The Agent page does not show this list. */
export const CrossRepoContext: Story = {
  render: () => (
    <div className="min-h-screen bg-sidebar dark:bg-background">
      <div className={cn(factorySettingsNarrowSectionBodyClassName, "pt-14")}>
        <CrossRepoContextCard
          canUpdate
          host="github"
          connected
          catalog={["superplanehq/suss", "superplanehq/runner", "superplanehq/superplane"]}
          initial={["superplanehq/suss", "superplanehq/runner"]}
        />
      </div>
    </div>
  ),
};

/** Outbound MCP list request fails. */
export const Error: Story = {
  render: () =>
    agentHarness({
      ...defaultFactoriesFixture,
      failAgentResourcesListForFactoryIds: [PRIMARY_FACTORY_ID],
    }),
};
