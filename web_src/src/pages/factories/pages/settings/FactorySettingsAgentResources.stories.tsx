import type { Meta, StoryObj } from "@storybook/react-vite";

import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { INLINE_SKILL, UI_UX_PRO_MAX_SKILL } from "../../__fixtures__/agentResourceFixtures";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayout } from "./FactorySettingsLayout";

/** Agent MCP stories live in `FactorySettingsMCPPage.stories.tsx`. */
const meta = {
  title: "Factories/Pages/Settings/Agent skills",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent`;

function skillsHarness(pathSuffix: string, resources: (typeof INLINE_SKILL)[]) {
  return (
    <FactoriesHarness
      pathSuffix={pathSuffix}
      experimentalFeatures={[FEATURE_WORKSPACE_SKILLS]}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        agentResourcesByFactoryId: {
          [PRIMARY_FACTORY_ID]: resources,
        },
      }}
    />
  );
}

export const Empty: Story = {
  render: () => skillsHarness(agentPath, []),
};

export const InlineSkill: Story = {
  render: () => skillsHarness(agentPath, [INLINE_SKILL]),
};

export const GitHubSkill: Story = {
  render: () => skillsHarness(agentPath, [UI_UX_PRO_MAX_SKILL]),
};

export const EditorNew: Story = {
  render: () => skillsHarness(`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/skills/new`, [INLINE_SKILL]),
};
