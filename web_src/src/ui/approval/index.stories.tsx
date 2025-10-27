import type { Meta, StoryObj } from '@storybook/react';
import { Approval, type ApprovalProps } from './';

const approveRelease: ApprovalProps = {
  title: "Approve Release",
  description: "New releases are deployed to staging for testing and require approvals.",
  iconSlug: "hand",
  iconColor: "text-orange-500",
  headerColor: "bg-orange-100",
  approvals: [
    {
      title: "Security",
      approved: false,
      interactive: true,
      requireArtifacts: [
        {
          label: "CVE Report",
        }
      ],
      onApprove: (artifacts) => console.log("Security approved with artifacts:", artifacts),
      onReject: (comment) => console.log("Security rejected with comment:", comment),
    },
    {
      title: "Compliance",
      approved: true,
      artifactCount: 1,
      artifacts: {
        "Security Audit Report": "https://example.com/audit-report.pdf",
        "Compliance Certificate": "https://example.com/cert.pdf",
      },
      href: "#",
    },
    {
      title: "Engineering",
      rejected: true,
      approverName: "Lucas Pinheiro",
      rejectionComment: "Security vulnerabilities need to be addressed before approval",
    },
    {
      title: "Josh Brown",
      approved: true,
    },
    {
      title: "Admin",
      approved: false,
    },
  ],
  awaitingEvent: {
    title: "fix: open rejected events tab",
    subtitle: "ef758d40",
  },
  receivedAt: new Date(new Date().getTime() - 1000 * 60 * 60 * 24),
}

const meta: Meta<typeof Approval> = {
  title: 'ui/Approval',
  component: Approval,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const ApprovalExpanded: Story = {
  args: approveRelease,
};

export const ApprovalCollapsed: Story = {
  args: {
    ...approveRelease,
    collapsed: true,
    collapsedBackground: "bg-orange-100",
  },
};

export const ApprovalZeroState: Story = {
  args: {
    title: "Approve Release",
    description: "New releases are deployed to staging for testing and require approvals.",
    iconSlug: "hand",
    iconColor: "text-orange-500",
    headerColor: "bg-orange-100",
    approvals: [],
    zeroStateText: "Waiting for events to require approval...",
  },
};