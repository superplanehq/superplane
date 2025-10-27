import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "@xyflow/react";

import "@xyflow/react/dist/style.css";
import "./../canvas-reset.css";

import githubIcon from "@/assets/icons/integrations/github.svg";

import { CanvasPage } from "./../index";
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
          { icon: "settings", items: ["TTL: 2d 15h"] },
          { icon: "settings", items: ["Max Envs: 50"] }
        ],
        metadata: [
          { icon: "server", label: "Active: 7/50 environments" },
          { icon: "triangle-alert", label: "Failed: 2" },
        ],
        lastRunItems: [
          {
            title: "Environment: env-4523",
            subtitle: "2d 14h left",
            childEventsInfo: {
              count: 1,
              state: "running",
              waitingInfos: [
                {
                  icon: "clock",
                  info: "Waiting for DNS propagation",
                  futureTimeDate: new Date(
                    new Date().getTime() + 1000 * 60 * 210
                  ), // 3h 30min = 210 minutes
                },
              ],
            },
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Sarah Chen",
              "Environment ID": "env-pr-4523",
              URL: "https://pr-4523.staging.app.com",
              "Shutdown at": new Date(
                new Date().getTime() + 1000 * 60 * 60 * 12
              ).toLocaleString(),
            },
          },
          {
            title: "Environment: env-4522",
            subtitle: "2d 2h left",
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Diego Morales",
              "Environment ID": "env-pr-4522",
              URL: "https://pr-4522.staging.app.com",
            },
          },
          {
            title: "Environment: env-4521",
            subtitle: "1d 19h left",
            state: "running",
            values: {
              "Triggered by": "PR comment",
              User: "Liam Patel",
              "Environment ID": "env-pr-4521",
              URL: "https://pr-4521.staging.app.com",
            },
          },
          {
            title: "Environment: env-4520",
            subtitle: "failed to start",
            state: "1d 12h left",
            values: {
              "Triggered by": "PR comment",
              User: "Ava Singh",
              "Environment ID": "env-pr-4520",
              "Failure reason": "Build artifact missing",
            },
          },
          {
            title: "Environment: env-4519",
            subtitle: "21h left",
            state: "running",
            values: {
              "Triggered by": "Pipeline",
              User: "System",
              "Environment ID": "env-pr-4519",
              URL: "https://pr-4519.staging.app.com",
            },
          },
        ],
        lastRunTotalCount: 7,
        startLastValuesOpen: true,
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
          childEventsInfo: {
            count: 2,
            state: "processed",
            waitingInfos: [],
          },
          state: "success",
          values: {
            "Environment ID": "env-pr-4501",
            "Triggered by": "System",
            "Resources removed": "Database, Storage, DNS, Network",
            "Duration": "2m 37s",
            "Completed at": new Date(
              new Date().getTime() - 1000 * 60 * 118
            ).toLocaleString(),
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
    ]
  },
  render: (args) => {
    return (
      <div className="h-[100vh] w-full ">
        <CanvasPage {...args} />
      </div>
    );
  },
};

Ephemeral.storyName = "03 - Ephemeral Environments";
