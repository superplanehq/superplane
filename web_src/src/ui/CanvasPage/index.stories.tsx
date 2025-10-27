import type { Meta, StoryObj } from "@storybook/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";
import { MainSubWorkflow, SubWorkflowsMap } from "./storybooks/subworkflows";

import { useEffect, useState } from "react";
import { CanvasPage } from "./index";
import {
  getStorybookData,
  isInStorybook,
  navigateToStory,
} from "./storybooks/navigation";

const meta = {
  title: "Pages/CanvasPage",
  component: CanvasPage,
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {},
} satisfies Meta<typeof CanvasPage>;

export default meta;

type Story = StoryObj<typeof CanvasPage>;

export const BlueprintExecutionPage: Story = {
  args: MainSubWorkflow,
  render: (args) => {
    // Get data passed from SimpleDeployment story (Storybook only)
    const [executionData, setExecutionData] = useState<any>(null);

    useEffect(() => {
      const data = getStorybookData();

      if (data) {
        setExecutionData(data);
      }
    }, []);

    // Use passed data to customize the story if available
    const dynamicTitle = executionData?.title || args.title;
    const subworkflowData = SubWorkflowsMap[executionData?.title];


    // Only override breadcrumbs if we have execution data, otherwise use args.breadcrumbs
    const dynamicBreadcrumbs = subworkflowData?.breadcrumbs
      ? subworkflowData?.breadcrumbs
      : args.breadcrumbs;

    const dynamicEdges = subworkflowData?.edges
      ? subworkflowData?.edges
      : args.edges;

    const dynamicNodes = subworkflowData?.nodes
      ? subworkflowData?.nodes
      : args.nodes;

    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage
          {...args}
          nodes={dynamicNodes}
          edges={dynamicEdges}
          title={dynamicTitle}
          breadcrumbs={dynamicBreadcrumbs}
        />
        {/* Debug info for Storybook (only visible in development) */}
        {isInStorybook() && executionData && (
          <div className="absolute top-16 right-4 z-30 bg-black/80 text-white p-3 rounded text-xs max-w-md">
            <div className="font-bold mb-2">
              📊 Execution Data (Storybook Only)
            </div>
            <div>From: {executionData.parentWorkflow}</div>
            <div>Node: {executionData.nodeId}</div>
            <div>Title: {executionData.title}</div>
            <div>
              Timestamp:{" "}
              {new Date(executionData.timestamp).toLocaleTimeString()}
            </div>
          </div>
        )}
      </div>
    );
  },
};
