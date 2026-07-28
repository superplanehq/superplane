import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";

import { activeWorkItemData } from "./__fixtures__/activeWorkItemData";
import { projectXWorkspaceData } from "./__fixtures__/workspaceData";
import { ActiveWorkItemPage } from "./ActiveWorkItemPage";
import { WorkspacePage } from "./index";
import type { CreateWorkRequest, WorkspacePageData } from "./types";

const meta = {
  title: "Pages/Workspace",
  component: WorkspacePage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof WorkspacePage>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Blank: Story = {
  render: () => <WorkspaceStory />,
};

export const ActiveWorkItem: Story = {
  render: () => (
    <OrgWorkspaceHarness
      startAt="workItem"
      appElement={<WorkspacePage data={projectXWorkspaceData} />}
      workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />}
    />
  ),
};

function WorkspaceStory() {
  const [data, setData] = useState<WorkspacePageData>(projectXWorkspaceData);

  const createWork = ({ title }: CreateWorkRequest) => {
    setData((current) => ({
      ...current,
      metrics: current.metrics.map((metric) =>
        metric.label === "Active work"
          ? { ...metric, value: String(current.workItems.length + 1), detail: "9 agents in flight" }
          : metric,
      ),
      workItems: [
        {
          id: "PX-129",
          title,
          stage: "Intake",
          branch: "Factory is preparing a branch",
          agentCount: 1,
          elapsed: "Now",
          status: "running",
          detail: "Factory is preparing the work order context.",
          updatedAt: "Now",
        },
        ...current.workItems,
      ],
    }));
  };

  return (
    <OrgWorkspaceHarness
      startAt="app"
      appElement={<WorkspacePage data={data} onCreateWork={createWork} />}
      workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />}
    />
  );
}
