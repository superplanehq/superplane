import type { Meta, StoryObj } from "@storybook/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";
import { MainSubWorkflow, SubWorkflowsMap } from "./storybooks/subworkflows";

import { useEffect, useMemo, useState } from "react";
import { CanvasPage, type CanvasNode } from "./index";
import type { BlockData } from "./Block";
import { createGetSidebarData } from "./storybooks/getSidebarData";
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
    const [nodes, setNodes] = useState<CanvasNode[]>([]);

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

    // Initialize local nodes state when dynamicNodes changes
    useEffect(() => {
      if (dynamicNodes) {
        console.log('Setting initial nodes:', dynamicNodes.length);
        setNodes(dynamicNodes);
      }
    }, [dynamicNodes]);

    const getSidebarData = useMemo(
      () => createGetSidebarData(nodes ?? []),
      [nodes]
    );

    const toggleNodeCollapse = (nodeId: string) => {
      console.log('toggleNodeCollapse called for nodeId:', nodeId);
      setNodes(prevNodes => {
        console.log('Current nodes:', prevNodes.length);
        const newNodes = prevNodes.map(node => {
          if (node.id !== nodeId) return node;

          console.log('Found node to toggle:', nodeId, node.data);
          const nodeData = { ...node.data } as unknown as BlockData;

          // Toggle collapse state based on node type
          if (nodeData.type === "composite" && nodeData.composite) {
            console.log('Toggling composite from', nodeData.composite.collapsed, 'to', !nodeData.composite.collapsed);
            nodeData.composite = {
              ...nodeData.composite,
              collapsed: !nodeData.composite.collapsed,
            };
          }

          if (nodeData.type === "approval" && nodeData.approval) {
            console.log('Toggling approval from', nodeData.approval.collapsed, 'to', !nodeData.approval.collapsed);
            nodeData.approval = {
              ...nodeData.approval,
              collapsed: !nodeData.approval.collapsed,
            };
          }

          if (nodeData.type === "trigger" && nodeData.trigger) {
            console.log('Toggling trigger from', nodeData.trigger.collapsed, 'to', !nodeData.trigger.collapsed);
            nodeData.trigger = {
              ...nodeData.trigger,
              collapsed: !nodeData.trigger.collapsed,
            };
          }

          const updatedNode: CanvasNode = { ...node, data: nodeData as unknown as Record<string, unknown> };
          console.log('Updated node:', updatedNode);
          return updatedNode;
        });
        console.log('Returning new nodes:', newNodes.length);
        return newNodes;
      });
    };

    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage
          {...args}
          nodes={nodes}
          edges={dynamicEdges}
          title={dynamicTitle}
          breadcrumbs={dynamicBreadcrumbs}
          getSidebarData={getSidebarData}
          onRun={(nodeId) => {
            console.log("Run action for node:", nodeId);
          }}
          onDuplicate={(nodeId) => {
            console.log("Duplicate action for node:", nodeId);
          }}
          onDocs={(nodeId) => {
            console.log("Documentation action for node:", nodeId);
          }}
          onToggleView={(nodeId) => {
            console.log("Toggle view action for node:", nodeId);
            console.log("Current nodes before toggle:", nodes.length);
            console.log("Node data before toggle:", nodes.find(n => n.id === nodeId)?.data);
            toggleNodeCollapse(nodeId);
          }}
          onDeactivate={(nodeId) => {
            console.log("Deactivate action for node:", nodeId);
          }}
          onDelete={(nodeId) => {
            console.log("Delete action for node:", nodeId);
          }}
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
