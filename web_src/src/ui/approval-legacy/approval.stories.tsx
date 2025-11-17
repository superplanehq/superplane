import type { Meta, StoryObj } from "@storybook/react";

import { Approval, type ApprovalProps } from "./index";

const meta = {
  title: "ui/ApprovalLegacy",
  component: Approval,
  tags: ["autodocs"],
  parameters: {
    layout: "centered",
  },
  argTypes: {
    title: {
      control: { type: "text" },
    },
    status: {
      control: { type: "text" },
      table: { disable: true },
    },
    version: {
      control: { type: "text" },
    },
    className: {
      control: { type: "text" },
      table: { disable: true },
    },
    approvals: {
      control: { type: "object" },
    },
    collapsed: {
      control: { type: "boolean" },
    },
    selected: {
      control: { type: "boolean" },
    },
    footerContent: {
      control: { disable: true },
      table: { disable: true },
    },
  },
} satisfies Meta<typeof Approval>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "Compliance Checker",
    status: "AWAITING APPROVAL",
    version: "superplane-1.0.9",
    selected: false,
    collapsed: false,
    approvals: [
      {
        title: "Security Team",
        approved: false,
        interactive: true,
        requireArtifacts: [
          {
            label: "CVE Report",
          },
          {
            label: "Security sign-off sheet",
            optional: true,
          },
        ],
        onApprove: (artifacts) => console.log("Security approved with artifacts:", artifacts),
        onReject: (comment) => console.log("Security rejected with comment:", comment),
      },
      {
        title: "Compliance Team",
        approved: true,
        approverName: "Petar P.",
        approverAvatar: "https://i.pravatar.cc/150?img=12",
        artifactCount: 2,
        artifacts: {
          "Security Audit Report": "https://example.com/audit-report.pdf",
          "Compliance Certificate": "https://example.com/cert.pdf",
        },
        href: "#",
      },
      {
        title: "Engineering Team",
        rejected: true,
        approverName: "Lucas P.",
        approverAvatar: "https://i.pravatar.cc/150?img=8",
        rejectionComment: "Security vulnerabilities need to be addressed before approval",
        href: "#",
      },
      {
        title: "Josh Brown",
        approved: true,
        href: "#",
      },
      {
        title: "Admin",
        approved: false,
        href: "#",
      },
    ] satisfies ApprovalProps["approvals"],
  },
};

export const ZeroState: Story = {
  args: {
    title: "Compliance Checker",
    status: "AWAITING APPROVAL",
    version: "superplane-1.0.9",
    approvals: [],
    selected: false,
  },
};

export const Collapsed: Story = {
  args: {
    title: "Compliance Checker",
    collapsed: true,
    selected: false,
    approvals: [
      {
        title: "Security",
        approved: false,
        href: "#",
      },
      {
        title: "Compliance",
        approved: true,
        artifactCount: 2,
        href: "#",
      },
      {
        title: "Alex Mitrovic",
        approved: false,
        href: "#",
      },
      {
        title: "Lucas Pinheiro",
        approved: false,
        href: "#",
      },
      {
        title: "Admin",
        approved: true,
        href: "#",
      },
    ] satisfies ApprovalProps["approvals"],
  },
};
