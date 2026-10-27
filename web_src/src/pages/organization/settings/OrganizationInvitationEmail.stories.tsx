import type { Meta, StoryObj } from "@storybook/react-vite";
import invitationTemplate from "../../../../../templates/email/organization_invitation.html?raw";

interface InvitationEmailArgs {
  organizationName: string;
  inviterName: string;
  role: string;
  expiresAt: string;
  previewWidth: number;
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function InvitationEmailPreview({ organizationName, inviterName, role, expiresAt, previewWidth }: InvitationEmailArgs) {
  const values: Record<string, string> = {
    OrganizationName: organizationName,
    InviterName: inviterName,
    Role: role,
    ExpiresAt: expiresAt,
    AcceptanceURL: "https://example.com/invitations/preview",
  };
  const html = invitationTemplate.replace(/\{\{\.(\w+)\}\}/g, (placeholder: string, field: string) => {
    const value = values[field];
    return value === undefined ? placeholder : escapeHtml(value);
  });

  return (
    <div style={{ padding: 24, minHeight: "100vh", background: "#f4f4f5" }}>
      <iframe
        title="Organization invitation email"
        srcDoc={html}
        sandbox=""
        referrerPolicy="no-referrer"
        style={{
          display: "block",
          width: previewWidth,
          maxWidth: "100%",
          height: 640,
          margin: "0 auto",
          border: "1px solid #d4d4d8",
          background: "#ffffff",
          colorScheme: "light",
        }}
      />
    </div>
  );
}

const meta = {
  title: "Emails/Organization invitation",
  component: InvitationEmailPreview,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Preview the current invitation email with sample values. The invitation link uses example.com.",
      },
    },
  },
  args: {
    organizationName: "Acme Engineering",
    inviterName: "Ana Petrović",
    role: "Operator",
    expiresAt: "Tue, 03 Nov 2026 12:00:00 UTC",
    previewWidth: 720,
  },
  argTypes: {
    organizationName: { control: "text", name: "Organization name" },
    inviterName: { control: "text", name: "Inviter name" },
    role: { control: "text", name: "Role" },
    expiresAt: { control: "text", name: "Expiration time" },
    previewWidth: { control: false, table: { disable: true } },
  },
} satisfies Meta<typeof InvitationEmailPreview>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Desktop: Story = {};

export const Mobile: Story = {
  args: { previewWidth: 375 },
};

export const LongOrganizationName: Story = {
  args: {
    organizationName: "Acme Research & Development, Infrastructure, and Reliability Engineering",
    previewWidth: 375,
  },
};
