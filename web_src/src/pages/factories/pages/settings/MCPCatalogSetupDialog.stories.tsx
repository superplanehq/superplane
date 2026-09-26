import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { MCPCatalogSetupDialog } from "./MCPCatalogSetupDialog";
import { MCP_CATALOG } from "./mcpCatalog";

const github = MCP_CATALOG.find((entry) => entry.id === "github");
const linear = MCP_CATALOG.find((entry) => entry.id === "linear");

const meta = {
  title: "Factories/Pages/Settings/MCP catalog setup",
  component: MCPCatalogSetupDialog,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof MCPCatalogSetupDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function HeaderSetup() {
  const [open, setOpen] = useState(true);
  return (
    <MCPCatalogSetupDialog
      open={open}
      entry={github}
      isSaving={false}
      onClose={() => setOpen(false)}
      onSignIn={async () => undefined}
      onSaveToken={async () => setOpen(false)}
    />
  );
}

function OAuthSetup() {
  const [open, setOpen] = useState(true);
  return (
    <MCPCatalogSetupDialog
      open={open}
      entry={linear}
      isSaving={false}
      onClose={() => setOpen(false)}
      onSignIn={async () => setOpen(false)}
      onSaveToken={async () => undefined}
    />
  );
}

export const GitHubToken: Story = {
  render: () => <HeaderSetup />,
};

export const LinearSignIn: Story = {
  render: () => <OAuthSetup />,
};
