import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";

import "@xyflow/react/dist/style.css";
import "../canvas-reset.css";

import { useCallback, useMemo, useState } from "react";
import type { BlockData } from "../Block";
import { CanvasPage, type CanvasNode } from "../index";
import { createGetSidebarData } from "./getSidebarData";

const meta = {
  title: "Pages/CanvasPage/Examples",
  component: CanvasPage,
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {},
} satisfies Meta<typeof CanvasPage>;

export default meta;

type Story = StoryObj<typeof CanvasPage>;

const sampleNodes: Node[] = [
  {
    id: "db-health",
    position: { x: 500, y: -800 },
    data: {
      label: "Database Health Monitor",
      state: "pending",
      type: "composite",
      composite: {
        title: "Database Health Monitor",
        description: "",
        iconSlug: "database",
        iconColor: "text-green-700",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-100",
        metadata: [
          { icon: "check-circle", label: "Connection: Healthy" },
          { icon: "clock", label: "Replication Lag: 45ms" },
          { icon: "zap", label: "Query Time: 12ms avg" },
          { icon: "activity", label: "Pool: 45/100 connections" },
        ],
        parameters: [
          {
            icon: "database",
            items: ["db-primary", "db-replica-1", "db-replica-2"],
          },
        ],
        lastRunItem: {
          title: "Database health check",
          subtitle: "45ms lag",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 6), // 6 minutes ago
          childEventsInfo: {
            count: 3,
            state: "processed",
            waitingInfos: [],
          },
          state: "success",
          values: {
            Connection: "Healthy",
            "Replication Lag": "45ms",
            "Avg Query Time": "12ms",
            "Pool Usage": "45/100",
          },
        },
        collapsed: false,
      },
    },
  },
  {
    id: "infra-monitor",
    position: { x: 0, y: -800 },
    data: {
      label: "Infrastructure Resource Monitor",
      state: "pending",
      type: "composite",
      composite: {
        title: "Infrastructure Resource Monitor",
        description: "",
        iconSlug: "cpu",
        iconColor: "text-green-700",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-100",
        metadata: [
          { icon: "cpu", label: "CPU: 45%" },
          { icon: "hard-drive", label: "Memory: 12.3 GB available" },
          { icon: "hard-drive", label: "Disk: 85% used" },
          { icon: "box", label: "Pods: 11/24 healthy" },
        ],
        parameters: [
          { icon: "server", items: ["prod-cluster-1", "prod-cluster-2"] },
        ],
        lastRunItem: {
          title: "Resource check",
          subtitle: "11/24 pods",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 8), // 8 minutes ago
          childEventsInfo: {
            count: 2,
            state: "processed",
            waitingInfos: [],
          },
          state: "failure",
          values: {
            "CPU Usage": "45%",
            Memory: "12.3 GB available",
            Disk: "85% used",
            Pods: "24/24",
          },
        },
        collapsed: false,
      },
    },
  },
  {
    id: "deploy-test",
    position: { x: -500, y: -800 },
    data: {
      label: "Traffic / Load Monitor",
      state: "pending",
      type: "composite",
      composite: {
        title: "Traffic / Load Monitor",
        description: "",
        iconSlug: "trending-up",
        iconColor: "text-green-700",
        headerColor: "bg-green-100",
        collapsedBackground: "bg-green-100",
        metadata: [
          { icon: "activity", label: "Requests/sec: 1,247 req/s" },
          { icon: "users", label: "Active Connections: 3,842" },
          { icon: "alert-circle", label: "Error Rate: 0.3%" },
          { icon: "server", label: "Load Balancer: Healthy" },
        ],
        parameters: [
          { icon: "map", items: ["us-west-1", "eu-global-1", "asia-east-1"] },
        ],
        lastRunItem: {
          title: "Traffic monitoring check",
          subtitle: "1,247 req/s",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 15), // 15 minutes ago
          childEventsInfo: {
            count: 3,
            state: "processed",
            waitingInfos: [],
          },
          state: "success",
          values: {
            "Requests/sec": "1,247",
            Connections: "3,842",
            "Error Rate": "0.3%",
            Status: "Healthy",
          },
        },
        collapsed: false,
      },
    },
  },
];

const sampleEdges: Edge[] = [
  { id: "e1", source: "listen-code", target: "build-stage" },
  { id: "e2", source: "listen-image", target: "build-stage" },
  { id: "e3", source: "build-stage", target: "approve" },
  { id: "e4", source: "approve", target: "deploy-us" },
  { id: "e5", source: "approve", target: "deploy-eu" },
  { id: "e6", source: "approve", target: "deploy-asia" },
];

export const Monitor: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
    title: "Infrastructure Monitoring",
  },
  render: function MonitorRender(args) {
    const [nodes, setNodes] = useState<CanvasNode[]>(args.nodes ?? []);
    const edges = useMemo(() => args.edges ?? [], [args.edges]);

    const toggleNodeCollapse = useCallback((nodeId: string) => {
      console.log("toggleNodeCollapse called for nodeId:", nodeId);
      setNodes((prevNodes) => {
        console.log("Current nodes:", prevNodes.length);
        const newNodes = prevNodes.map((node) => {
          if (node.id !== nodeId) return node;

          console.log("Found node to toggle:", nodeId, node.data);
          const nodeData = { ...node.data } as unknown as BlockData;

          // Toggle collapse state based on node type
          if (nodeData.type === "composite" && nodeData.composite) {
            console.log(
              "Toggling composite from",
              nodeData.composite.collapsed,
              "to",
              !nodeData.composite.collapsed
            );
            nodeData.composite = {
              ...nodeData.composite,
              collapsed: !nodeData.composite.collapsed,
            };
          }

          if (nodeData.type === "approval" && nodeData.approval) {
            console.log(
              "Toggling approval from",
              nodeData.approval.collapsed,
              "to",
              !nodeData.approval.collapsed
            );
            nodeData.approval = {
              ...nodeData.approval,
              collapsed: !nodeData.approval.collapsed,
            };
          }

          if (nodeData.type === "trigger" && nodeData.trigger) {
            console.log(
              "Toggling trigger from",
              nodeData.trigger.collapsed,
              "to",
              !nodeData.trigger.collapsed
            );
            nodeData.trigger = {
              ...nodeData.trigger,
              collapsed: !nodeData.trigger.collapsed,
            };
          }

          const updatedNode: CanvasNode = {
            ...node,
            data: nodeData as unknown as Record<string, unknown>,
          };
          console.log("Updated node:", updatedNode);
          return updatedNode;
        });
        console.log("Returning new nodes:", newNodes.length);
        return newNodes;
      });
    }, []);

    const getSidebarData = useMemo(
      () => createGetSidebarData(nodes ?? []),
      [nodes]
    );

    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage
          {...args}
          nodes={nodes}
          edges={edges}
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
          onEdit={(nodeId) => {
            console.log("Edit action for node:", nodeId);
          }}
          onToggleView={(nodeId) => {
            console.log("Toggle view action for node:", nodeId);
            console.log("Current nodes before toggle:", nodes.length);
            console.log(
              "Node data before toggle:",
              nodes.find((n) => n.id === nodeId)?.data
            );
            toggleNodeCollapse(nodeId);
          }}
          onDeactivate={(nodeId) => {
            console.log("Deactivate action for node:", nodeId);
          }}
          onDelete={(nodeId) => {
            console.log("Delete action for node:", nodeId);
          }}
        />
      </div>
    );
  },
};

Monitor.storyName = "02 - Inftrastructure Monitoring";
