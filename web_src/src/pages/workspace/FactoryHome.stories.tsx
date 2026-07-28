import type { Meta, StoryObj } from "@storybook/react-vite";

import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";

import { activeWorkItemData } from "./__fixtures__/activeWorkItemData";
import { projectXWorkspaceData } from "./__fixtures__/workspaceData";
import { ActiveWorkItemPage } from "./ActiveWorkItemPage";
import { WorkspacePage } from "./index";

const meta = {
  title: "Pages/Factory",
  component: WorkspacePage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof WorkspacePage>;

export default meta;

type Story = StoryObj<typeof meta>;

function FactoryStory({ defaultTab }: { defaultTab: "overview" | "work-orders" | "automations" | "velocity" }) {
  return (
    <OrgWorkspaceHarness
      startAt="app"
      appElement={<WorkspacePage data={projectXWorkspaceData} defaultTab={defaultTab} />}
      workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />}
    />
  );
}

export const Overview: Story = {
  render: () => <FactoryStory defaultTab="overview" />,
};

export const WorkOrders: Story = {
  render: () => <FactoryStory defaultTab="work-orders" />,
};

export const Automations: Story = {
  render: () => <FactoryStory defaultTab="automations" />,
};

export const Velocity: Story = {
  render: () => <FactoryStory defaultTab="velocity" />,
};

export const WorkOrder: Story = {
  render: () => (
    <OrgWorkspaceHarness
      startAt="workItem"
      appElement={<WorkspacePage data={projectXWorkspaceData} />}
      workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />}
    />
  ),
};
