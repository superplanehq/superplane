import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router-dom";
import { RubricWidget } from "./RubricWidget";

const meta: Meta<typeof RubricWidget> = {
  title: "AgentSidebar/RubricWidget",
  component: RubricWidget,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => (
      <div className="max-w-md bg-white border border-slate-200 rounded-lg p-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof RubricWidget>;

export const AgentLinksAndTables: Story = {
  args: {
    title: "Build Plan",
    criteria: [
      {
        text: "Open the [run link](run:123e4567-e89b-12d3-a456-426614174000) and confirm it's preserved.",
      },
      {
        text: ["| Step | Owner |", "| --- | --- |", "| Generate OpenAPI client | Platform |"].join("\n"),
      },
      {
        text: "Ship it.",
      },
    ],
  },
};

export const InAppRunChip: Story = {
  decorators: [
    (Story) => (
      <MemoryRouter>
        <div className="max-w-md bg-white border border-slate-200 rounded-lg p-4">
          <Story />
        </div>
      </MemoryRouter>
    ),
  ],
  args: {
    title: "Build Plan",
    canvasId: "canvas_123",
    organizationId: "org_123",
    criteria: [
      {
        text: "Open the [run link](run:123e4567-e89b-12d3-a456-426614174000) (should render as an in-app chip).",
      },
    ],
  },
};
