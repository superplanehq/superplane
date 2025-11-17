import type { Meta, StoryObj } from "@storybook/react";

import { ApprovalItem } from "./index";
import { ItemGroup, ItemSeparator } from "../item";

const meta = {
  title: "ui/ApprovalItem",
  component: ApprovalItem,
  tags: ["autodocs"],
  parameters: {
    layout: "centered",
  },
  argTypes: {
    title: {
      control: { type: "text" },
    },
    approved: {
      control: { type: "boolean" },
    },
    href: {
      control: { type: "text" },
    },
    className: {
      control: { type: "text" },
    },
    interactive: {
      control: { type: "boolean" },
    },
  },
} satisfies Meta<typeof ApprovalItem>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    title: "Compliance",
    approved: true,
    href: "#",
  },
};

export const Pending: Story = {
  args: {
    title: "Product Lead",
    approved: false,
    href: "#",
  },
};

export const WithAvatar: Story = {
  args: {
    title: "Alex Mitrovic",
    approved: true,
    approverName: "Alex M.",
    approverAvatar: "https://i.pravatar.cc/150?img=12",
    href: "#",
  },
};

export const Interactive: Story = {
  args: {
    title: "Security",
    approved: false,
    interactive: true,
  },
  render: (args) => (
    <div className="w-full max-w-3xl">
      <ApprovalItem {...args} onApprove={() => console.log("Approved!")} onReject={() => console.log("Rejected!")} />
    </div>
  ),
};

export const FourItems: Story = {
  args: {
    title: "Compliance",
    approved: true,
  },
  render: () => (
    <ItemGroup className="w-full max-w-3xl">
      <ApprovalItem title="Compliance" approved={true} href="#" />
      <ItemSeparator />
      <ApprovalItem title="Product Lead" approved={false} href="#" />
      <ItemSeparator />
      <ApprovalItem title="Engineering Lead" approved={false} href="#" />
      <ItemSeparator />
      <ApprovalItem title="Security Team" approved={true} href="#" />
    </ItemGroup>
  ),
};

export const WithAvatars: Story = {
  args: {
    title: "Compliance",
    approved: true,
  },
  render: () => (
    <ItemGroup className="w-full max-w-3xl">
      <ApprovalItem
        title="Compliance"
        approved={true}
        approverName="Sarah K."
        approverAvatar="https://i.pravatar.cc/150?img=5"
        href="#"
      />
      <ItemSeparator />
      <ApprovalItem title="Product Lead" approved={false} href="#" />
      <ItemSeparator />
      <ApprovalItem title="Engineering Lead" approved={false} href="#" />
      <ItemSeparator />
      <ApprovalItem
        title="Alex Mitrovic"
        approved={true}
        approverName="Alex M."
        approverAvatar="https://i.pravatar.cc/150?img=12"
        href="#"
      />
    </ItemGroup>
  ),
};

export const InteractiveFourItems: Story = {
  args: {
    title: "Compliance",
    approved: true,
  },
  render: () => (
    <ItemGroup className="w-full max-w-3xl">
      <ApprovalItem
        title="Compliance"
        approved={true}
        interactive={true}
        onApprove={() => console.log("Compliance Approved!")}
        onReject={() => console.log("Compliance Rejected!")}
      />
      <ItemSeparator />
      <ApprovalItem
        title="Product Lead"
        approved={false}
        interactive={true}
        onApprove={() => console.log("Product Lead Approved!")}
        onReject={() => console.log("Product Lead Rejected!")}
      />
      <ItemSeparator />
      <ApprovalItem
        title="Engineering Lead"
        approved={false}
        interactive={true}
        onApprove={() => console.log("Engineering Lead Approved!")}
        onReject={() => console.log("Engineering Lead Rejected!")}
      />
      <ItemSeparator />
      <ApprovalItem
        title="Security"
        approved={false}
        interactive={true}
        onApprove={() => console.log("Security Approved!")}
        onReject={() => console.log("Security Rejected!")}
      />
    </ItemGroup>
  ),
};
