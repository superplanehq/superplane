import type { Meta, StoryObj } from "@storybook/react-vite";

import { OrgSsoSection } from "./OrgSsoSection";

const meta = {
  title: "Admin/Single Sign-On",
  component: OrgSsoSection,
} satisfies Meta<typeof OrgSsoSection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Connected: Story = {
  args: {
    orgId: "org",
    preview: {
      connection: { exists: true, status: "active", enforcement: "required" },
      domains: [{ id: "domain", domain: "example.com", verified: false }],
    },
  },
};

export const LoadError: Story = {
  args: {
    orgId: "org",
    preview: {
      connection: null,
      domains: [],
      error: "Single Sign-On details are not available.",
    },
  },
};
