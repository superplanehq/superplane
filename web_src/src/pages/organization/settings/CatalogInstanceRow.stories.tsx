import type { Meta, StoryObj } from "@storybook/react-vite";

import type { OrganizationsIntegration } from "@/api-client";
import { GITHUB_APP_INSTALLATION_URL_PROPERTY_NAME } from "@/lib/integrations";
import { catalogAppearance } from "./integrationCatalogAppearance";
import { CatalogInstanceRow } from "./CatalogInstanceRow";

const INSTALLATION_URL = "https://github.com/organizations/acme/settings/installations/123";

const CONNECTED_INSTANCE: OrganizationsIntegration = {
  metadata: { id: "integration-1", name: "github-prod", integrationName: "github" },
  status: {
    state: "ready",
    properties: [{ name: GITHUB_APP_INSTALLATION_URL_PROPERTY_NAME, value: INSTALLATION_URL }],
  },
};

const INSTANCE_WITHOUT_INSTALLATION_LINK: OrganizationsIntegration = {
  metadata: { id: "integration-2", name: "github-staging", integrationName: "github" },
  status: { state: "ready", properties: [] },
};

const meta = {
  title: "Organization/Settings/CatalogInstanceRow",
  component: CatalogInstanceRow,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <div className="max-w-xl rounded-md border border-border bg-card p-2">
        <Story />
      </div>
    ),
  ],
  args: {
    appearance: "factories",
    styles: catalogAppearance("factories"),
    canUpdateIntegrations: true,
    permissionsLoading: false,
    onConfigure: () => console.log("configure"),
  },
} satisfies Meta<typeof CatalogInstanceRow>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The hosted GitHub App stored an installation URL. The row shows both actions. */
export const WithInstallationLink: Story = {
  args: {
    integration: CONNECTED_INSTANCE,
  },
};

/** Older instances, or providers other than the hosted GitHub App, have no installation URL. */
export const WithoutInstallationLink: Story = {
  args: {
    integration: INSTANCE_WITHOUT_INSTALLATION_LINK,
  },
};

/** A member without the update-integrations permission sees a disabled Configure button and a tooltip. */
export const PermissionDenied: Story = {
  args: {
    integration: CONNECTED_INSTANCE,
    canUpdateIntegrations: false,
  },
};

/** The legacy organization settings page renders the same row with its own visual style. */
export const LegacyAppearance: Story = {
  args: {
    appearance: "legacy",
    styles: catalogAppearance("legacy"),
    integration: CONNECTED_INSTANCE,
  },
};
