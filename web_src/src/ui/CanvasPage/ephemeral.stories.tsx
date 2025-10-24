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
          title: "feat: add ephemeral environments",
          sizeInMB: 2.3,
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
];

const ephemeralEdges: Edge[] = [];

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
