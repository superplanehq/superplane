import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { MCPCatalogSetupDialog } from "./MCPCatalogSetupDialog";
import { MCP_CATALOG } from "./mcpCatalog";

const github = MCP_CATALOG.find((entry) => entry.id === "github");
const linear = MCP_CATALOG.find((entry) => entry.id === "linear");
const sentry = MCP_CATALOG.find((entry) => entry.id === "sentry");

const meta = {
  title: "Factories/Pages/Settings/MCP catalog setup",
  component: MCPCatalogSetupDialog,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof MCPCatalogSetupDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

function logAction(label: string, ...args: unknown[]) {
  // eslint-disable-next-line no-console
  console.log(label, ...args);
}

function HeaderSetup() {
  const [open, setOpen] = useState(true);
  return (
    <MCPCatalogSetupDialog
      open={open}
      entry={github}
      isSaving={false}
      onClose={() => {
        logAction("close GitHub setup");
        setOpen(false);
      }}
      onSignIn={async (entry) => {
        logAction("sign in", entry.id);
      }}
      onSaveToken={async (entry) => {
        logAction("save token", entry.id);
        setOpen(false);
      }}
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
      onClose={() => {
        logAction("close Linear setup");
        setOpen(false);
      }}
      onSignIn={async (entry) => {
        logAction("sign in", entry.id);
        setOpen(false);
      }}
      onSaveToken={async (entry) => {
        logAction("save token", entry.id);
      }}
    />
  );
}

export const GitHubToken: Story = {
  render: () => <HeaderSetup />,
};

export const LinearSignIn: Story = {
  render: () => <OAuthSetup />,
};

function SentrySetup() {
  const [open, setOpen] = useState(true);
  return (
    <MCPCatalogSetupDialog
      open={open}
      entry={sentry}
      isSaving={false}
      onClose={() => {
        logAction("close Sentry setup");
        setOpen(false);
      }}
      onSignIn={async (entry) => {
        logAction("sign in", entry.id);
        setOpen(false);
      }}
      onSaveToken={async (entry) => {
        logAction("save token", entry.id);
      }}
    />
  );
}

export const SentrySignIn: Story = {
  render: () => <SentrySetup />,
};
