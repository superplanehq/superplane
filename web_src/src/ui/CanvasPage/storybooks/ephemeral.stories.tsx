import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";

import "@xyflow/react/dist/style.css";
import "./../canvas-reset.css";

import githubIcon from "@/assets/icons/integrations/github.svg";

import { useCallback, useMemo, useState } from "react";
import type { BlockData } from "./../Block";
import { CanvasPage, type CanvasNode } from "./../index";
import { mockBuildingBlockCategories } from "./buildingBlocks";
import { createGetSidebarData } from "./getSidebarData";
import { handleNodeExpand } from "./navigation";

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

const ephemeralNodes: Node[] = [
  {
    id: "github-trigger",
    position: { x: 0, y: 0 },
    data: {
      label: "Listen to code changes",
      state: "working",
      type: "trigger",
      trigger: {
        title: "GitHub",
        iconSrc: githubIcon,
        iconBackground: "bg-black",
        headerColor: "bg-white",
        collapsedBackground: "bg-black",
        metadata: [
          { icon: "book", label: "monarch-app" },
          { icon: "filter", label: "issue_comment" },
        ],
        lastEventData: {
          title: "feat: arrange pets by cuteness",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 5), // 5 minutes ago
          state: "triggered",
        },
        collapsed: false,
      },
    },
  },
  {
    id: "manual-trigger",
    position: { x: 0, y: 400 },
    data: {
      label: "Manual run trigger",
      state: "pending",
      type: "trigger",
      trigger: {
        title: "Provision/Deprovision Environment",
        iconSlug: "play",
        iconColor: "text-purple-700",
        headerColor: "bg-purple-100",
        collapsedBackground: "bg-purple-100",
        metadata: [
          {
            icon: "chevrons-left-right-ellipsis",
            label: "Payload templates: 2",
          },
        ],
        lastEventData: {
          title: "Deprovision env-pr-4498",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 45), // 45 minutes ago
          state: "triggered",
        },
        collapsed: false,
      },
    },
  },
  {
    id: "ephemeral-provisioner",
    position: { x: 600, y: 0 },
    data: {
      label: "Provisioner",
      state: "pending",
      type: "composite",
      composite: {
        title: "Provisioner",
        iconSlug: "boxes",
        iconColor: "text-blue-700",
        headerColor: "bg-blue-100",
        collapsedBackground: "bg-blue-100",
        parameters: [
          { icon: "settings", items: { TTL: "2d 15h" } },
          { icon: "settings", items: { "Max Envs": "50" } },
        ],
        metadata: [
          { icon: "server", label: "Active: 7/50 environments" },
          { icon: "triangle-alert", label: "Failed: 2" },
        ],
        lastRunItems: [
          {
            title: "Environment: env-4523",
            subtitle: "2d 14h left",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 3), // 3 minutes ago
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for DNS propagation",
                  futureTimeDate: new Date(new Date().getTime() + 1000 * 60 * 210), // 3h 30min = 210 minutes
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Sarah Chen",
              "Environment ID": "env-pr-4523",
              URL: "https://pr-4523.staging.app.com",
              "Shutdown at": new Date(new Date().getTime() + 1000 * 60 * 60 * 12).toLocaleString(),
            },
          },
          {
            title: "Environment: env-4522",
            subtitle: "1d 8h left",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 45), // 45 minutes ago
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for health check",
                  futureTimeDate: new Date(new Date().getTime() + 1000 * 60 * 180), // 3 hours
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Alex Johnson",
              "Environment ID": "env-pr-4522",
              URL: "https://pr-4522.staging.app.com",
            },
          },
          {
            title: "Environment: env-4521",
            subtitle: "18h left",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 90), // 1.5 hours ago
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for SSL certificate",
                  futureTimeDate: new Date(new Date().getTime() + 1000 * 60 * 150), // 2.5 hours
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Maria Garcia",
              "Environment ID": "env-pr-4521",
              URL: "https://pr-4521.staging.app.com",
            },
          },
          {
            title: "Environment: env-4520",
            subtitle: "12h left",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 180), // 3 hours ago
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for database migration",
                  futureTimeDate: new Date(new Date().getTime() + 1000 * 60 * 120), // 2 hours
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "John Smith",
              "Environment ID": "env-pr-4520",
              URL: "https://pr-4520.staging.app.com",
            },
          },
          {
            title: "Environment: env-4519",
            subtitle: "6h left",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 240), // 4 hours ago
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for container startup",
                  futureTimeDate: new Date(new Date().getTime() + 1000 * 60 * 90), // 1.5 hours
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Emma Wilson",
              "Environment ID": "env-pr-4519",
              URL: "https://pr-4519.staging.app.com",
            },
          },
          {
            title: "Environment: env-4518",
            subtitle: "3h left",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 300), // 5 hours ago
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for load balancer",
                  futureTimeDate: new Date(new Date().getTime() + 1000 * 60 * 60), // 1 hour
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "David Lee",
              "Environment ID": "env-pr-4518",
              URL: "https://pr-4518.staging.app.com",
            },
          },
          {
            title: "Environment: env-4517",
            subtitle: "1h left",
            receivedAt: new Date(new Date().getTime() - 1000 * 60 * 360), // 6 hours ago
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for network setup",
                  futureTimeDate: new Date(new Date().getTime() + 1000 * 60 * 30), // 30 minutes
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Lisa Brown",
              "Environment ID": "env-pr-4517",
              URL: "https://pr-4517.staging.app.com",
            },
          },
        ],
        maxVisibleEvents: 5,
        collapsed: false,
      },
    },
  },
  {
    id: "ephemeral-deprovisioner",
    position: { x: 1200, y: 0 },
    data: {
      label: "Desprovisioner",
      state: "pending",
      type: "composite",
      composite: {
        title: "Deprovisioner",
        iconSlug: "trash-2",
        iconColor: "text-red-700",
        headerColor: "bg-red-100",
        collapsedBackground: "bg-red-100",
        metadata: [
          { icon: "trash", label: "Cleaned: 127 environments" },
          { icon: "check-circle", label: "Success: 98%" },
          { icon: "clock", label: "Avg time: 2m 45s" },
          { icon: "calendar", label: "Last: 2h ago" },
        ],
        lastRunItem: {
          title: "Environment: env-pr-4501",
          subtitle: "Expired",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 120), // 2 hours ago
          childEventsInfo: {
            count: 2,
            state: "triggered",
            waitingInfos: [],
          },
          state: "success",
          values: {
            "Environment ID": "env-pr-4501",
            "Triggered by": "System",
            "Resources removed": "Database, Storage, DNS, Network",
            Duration: "2m 37s",
            "Completed at": new Date(new Date().getTime() - 1000 * 60 * 118).toLocaleString(),
          },
        },
        collapsed: false,
      },
    },
  },
];

const ephemeralEdges: Edge[] = [
  { id: "e1", source: "github-trigger", target: "ephemeral-provisioner" },
  { id: "e2", source: "manual-trigger", target: "ephemeral-provisioner" },
  { id: "e3", source: "ephemeral-provisioner", target: "ephemeral-deprovisioner" },
  { id: "e4", source: "manual-trigger", target: "ephemeral-deprovisioner" },
];

export const Ephemeral: Story = {
  args: {
    nodes: ephemeralNodes,
    edges: ephemeralEdges,
    onNodeExpand: handleNodeExpand,
    breadcrumbs: [
      {
        label: "Workflows",
      },
      {
        label: "Ephemeral Environments",
      },
    ],
    buildingBlocks: mockBuildingBlockCategories,
  },
  render: (args) => {
    const [nodes, setNodes] = useState<CanvasNode[]>(args.nodes ?? []);

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
            console.log("Toggling composite from", nodeData.composite.collapsed, "to", !nodeData.composite.collapsed);
            nodeData.composite = {
              ...nodeData.composite,
              collapsed: !nodeData.composite.collapsed,
            };
          }

          if (nodeData.type === "component" && nodeData.component) {
            console.log("Toggling component from", nodeData.component.collapsed, "to", !nodeData.component.collapsed);
            nodeData.component = {
              ...nodeData.component,
              collapsed: !nodeData.component.collapsed,
            };
          }

          if (nodeData.type === "trigger" && nodeData.trigger) {
            console.log("Toggling trigger from", nodeData.trigger.collapsed, "to", !nodeData.trigger.collapsed);
            nodeData.trigger = {
              ...nodeData.trigger,
              collapsed: !nodeData.trigger.collapsed,
            };
          }

          const updatedNode: CanvasNode = { ...node, data: nodeData as unknown as Record<string, unknown> };
          console.log("Updated node:", updatedNode);
          return updatedNode;
        });
        console.log("Returning new nodes:", newNodes.length);
        return newNodes;
      });
    }, []);

    const getSidebarData = useMemo(() => createGetSidebarData(nodes ?? []), [nodes]);

    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage
          {...args}
          nodes={nodes}
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
            console.log("Node data before toggle:", nodes.find((n) => n.id === nodeId)?.data);
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

Ephemeral.storyName = "03 - Ephemeral Environments";
