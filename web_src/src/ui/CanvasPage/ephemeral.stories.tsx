import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import "./canvas-reset.css";

import githubIcon from "@/assets/icons/integrations/github.svg";

import { CanvasPage } from "./index";

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
        headerColor: "bg-gray-100",
        collapsedBackground: "bg-black",
        metadata: [
          { icon: "book", label: "monarch-app" },
          { icon: "filter", label: "issue_comment" },
        ],
        lastEventData: {
          title: "feat: arrange pets by cuteness",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 5), // 5 minutes ago
          state: "processed",
        },
        collapsed: false,
      },
    },
  },
  {
    id: "manual-trigger",
    position: { x: 0, y: 300 },
    data: {
      label: "Manual run trigger",
      state: "pending",
      type: "trigger",
      trigger: {
        title: "Provision Environment",
        iconSlug: "play",
        iconColor: "text-purple-700",
        headerColor: "bg-purple-100",
        collapsedBackground: "bg-purple-100",
        metadata: [
          { icon: "chevrons-left-right-ellipsis", label: "Payload templates: 1" },
        ],
        lastEventData: {
          title: "Manual deployment request",
          receivedAt: new Date(new Date().getTime() - 1000 * 60 * 45), // 45 minutes ago
          state: "processed",
        },
        collapsed: false,
      },
    },
  },
  {
    id: "ephemeral-provisioner",
    position: { x: 600, y: 0 },
    data: {
      label: "Ephemeral Environments Provisioner",
      state: "pending",
      type: "composite",
      composite: {
        title: "Ephemeral Environments Provisioner",
        description: "Provision and manage temporary test environments",
        iconSlug: "boxes",
        iconColor: "text-blue-700",
        headerColor: "bg-blue-100",
        collapsedBackground: "bg-blue-100",
        metadata: [
          { icon: "server", label: "Active: 7/50 environments" },
          { icon: "triangle-alert", label: "Failed: 2" },
        ],
        parameters: ["TTL: 12 hours", "Max Envs: 50"],
        parametersIcon: "settings",
        lastRunItem: {
          title: "Environment: env-4523",
          subtitle: "pr-213",
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
            "User": "Sarah Chen",
            "Environment ID": "env-pr-4523",
            "URL": "https://pr-4523.staging.app.com",
            "Shutdown at": new Date(new Date().getTime() + 1000 * 60 * 60 * 12).toLocaleString(),
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
];

export const Ephemeral: Story = {
  args: {
    nodes: ephemeralNodes,
    edges: ephemeralEdges,
  },
  render: (args) => {
    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage {...args} />
      </div>
    );
  },
};
