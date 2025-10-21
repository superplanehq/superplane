import type { Meta, StoryObj } from "@storybook/react"
import type { NodeProps } from "reactflow"

import githubIcon from "@/assets/icons/integrations/github.svg"
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg"
import scheduleIcon from "@/assets/icons/schedule.svg"

import { EventSource, type EventSourceProps } from "../eventSource"
import { Canvas } from "./index"

const EventSourceNode: React.FC<NodeProps<EventSourceProps & { collapsed?: boolean }>> = ({
  data,
  selected,
}) => (
  <EventSource
    {...data}
    selected={selected ?? data.selected ?? false}
    collapsed={data.collapsed ?? false}
  />
)

const meta = {
  title: "components/Canvas",
  component: Canvas,
  tags: ["autodocs"],
  parameters: {
    layout: "fullscreen",
  },
  argTypes: {
    containerClassName: {
      control: { type: "text" },
      table: { disable: true },
    },
    containerStyle: {
      control: { disable: true },
      table: { disable: true },
    },
    nodes: {
      control: { type: "object" },
    },
    edges: {
      control: { type: "object" },
    },
    nodeTypes: {
      control: { disable: true },
      table: { disable: true },
    },
    showControls: {
      control: { type: "boolean" },
      table: { category: "Display" },
    },
    showMiniMap: {
      control: { type: "boolean" },
      table: { category: "Display" },
    },
  },
} satisfies Meta<typeof Canvas>

export default meta

type Story = StoryObj<typeof Canvas>

export const Default: Story = {
  args: {
    containerClassName: "h-[600px]",
    fitView: false,
    nodes: [
      {
        id: "github-source",
        type: "eventSource",
        position: { x: 100, y: 60 },
        data: {
          title: "GitHub Source",
          content: "",
          resource: {
            label: "superplane-ui",
            href: "https://github.com/superplanehq/superplane-ui",
            icon: "book-marked",
          },
          badgeTone: "bg-gray-950",
          badgeLabel: "GitHub",
          badgeImageSrc: githubIcon,
          filters: ["branch = main", "status = success"],
          eventType: { label: "push" },
          events: [
            {
              status: "success",
              title: "fix: open rejected events tab when the sidebar is already open",
              timestamp: "2m ago",
              badges: [{ label: "push" }],
              href: "#",
            },
          ],
          collapsed: false,
          selected: true,
        } satisfies EventSourceProps & { collapsed: boolean },
      },
      {
        id: "pagerduty-source",
        type: "eventSource",
        position: { x: 100, y: 370 },
        data: {
          title: "PagerDuty Incidents",
          content: "",
          resource: {
            label: "On-call schedule",
            href: "https://pagerduty.com",
            icon: "clipboard-clock",
          },
          badgeTone: "bg-green-500",
          badgeLabel: "PagerDuty",
          badgeImageSrc: pagerDutyIcon,
          filters: ["urgency = high"],
          eventType: { label: "incident" },
          events: [],
          collapsed: true,
          selected: false,
        } satisfies EventSourceProps & { collapsed: boolean },
      },
    ],
    edges: [],
    nodeTypes: {
      eventSource: EventSourceNode,
    },
    showControls: true,
    showMiniMap: false,
  },
  render: (args) => <Canvas {...args} />,
}