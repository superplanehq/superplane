import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";
import { InvitationCapacityNotice } from "./OrganizationInvitations";

const meta = {
  title: "Organization/Settings/Members/Invitation capacity notice",
  component: InvitationCapacityNotice,
  decorators: [
    (Story) => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
  args: {
    organizationId: "org-1",
    manager: { canInvite: true, canManageBilling: true, canPurchaseSeats: true, capacity: { state: "available" } },
  },
} satisfies Meta<typeof InvitationCapacityNotice>;
export default meta;
type Story = StoryObj<typeof meta>;

export const PaymentBlocked: Story = {
  args: {
    manager: {
      canInvite: true,
      canManageBilling: true,
      canPurchaseSeats: false,
      capacity: { state: "paymentBlocked" },
    },
  },
};

export const PaymentBlockedWithoutBillingPermission: Story = {
  args: {
    manager: {
      canInvite: true,
      canManageBilling: false,
      canPurchaseSeats: false,
      capacity: { state: "paymentBlocked" },
    },
  },
};

export const PurchaseRequired: Story = {
  args: {
    manager: {
      canInvite: true,
      canManageBilling: true,
      canPurchaseSeats: true,
      capacity: { state: "purchaseRequired" },
    },
  },
};

export const PurchaseRequiredWithoutBillingPermission: Story = {
  args: {
    manager: {
      canInvite: true,
      canManageBilling: false,
      canPurchaseSeats: false,
      capacity: { state: "purchaseRequired" },
    },
  },
};

export const PlanMaximum: Story = {
  args: {
    manager: { canInvite: true, canManageBilling: true, canPurchaseSeats: false, capacity: { state: "planMaximum" } },
  },
};

export const OverCapacity: Story = {
  args: {
    manager: {
      canInvite: true,
      canManageBilling: true,
      canPurchaseSeats: true,
      capacity: { state: "overCapacity", members: 12, paidSeats: 10 },
    },
  },
};

export const MemberLimit: Story = {
  args: {
    manager: { canInvite: true, canManageBilling: true, canPurchaseSeats: false, capacity: { state: "memberLimit" } },
  },
};
