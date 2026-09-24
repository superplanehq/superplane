import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { MCPAddPicker } from "./MCPAddPicker";

const meta = {
  title: "Factories/Pages/Settings/MCP catalog",
  component: MCPAddPicker,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof MCPAddPicker>;

export default meta;

type Story = StoryObj<typeof meta>;

function OpenPicker() {
  const [open, setOpen] = useState(true);
  return <MCPAddPicker open={open} onClose={() => setOpen(false)} onSelect={() => setOpen(false)} />;
}

export const CatalogPicker: Story = {
  render: () => <OpenPicker />,
};
