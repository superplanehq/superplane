import type { Meta, StoryObj } from "@storybook/react";
import type { Edge, Node } from "reactflow";

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

const sampleNodes: Node[] = [
  {
    id: "listen-code",
    position: { x: 60, y: 80 },
    data: { label: "Listen to code changes" },
    type: "input",
  },
  {
    id: "listen-image",
    position: { x: 60, y: 260 },
    data: { label: "Listen to Docker image updates" },
    type: "input",
  },
  {
    id: "build-stage",
    position: { x: 320, y: 150 },
    data: { label: "Build/Test/Deploy to Stage" },
  },
  {
    id: "approve",
    position: { x: 620, y: 150 },
    data: { label: "Approve release" },
  },
  {
    id: "deploy-us",
    position: { x: 940, y: 40 },
    data: { label: "Deploy to US" },
    type: "output",
  },
  {
    id: "deploy-eu",
    position: { x: 940, y: 180 },
    data: { label: "Deploy to EU" },
    type: "output",
  },
  {
    id: "deploy-asia",
    position: { x: 940, y: 320 },
    data: { label: "Deploy to Asia" },
    type: "output",
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

export const SimpleDeployment: Story = {
  args: {
    nodes: sampleNodes,
    edges: sampleEdges,
  },
  render: (args) => <CanvasPage {...args} />,
};
