import type { Meta, StoryObj } from "@storybook/react-vite";
import { BillingPaymentFailedBanner } from "./BillingPaymentFailedBanner";

const meta = {
  title: "Factories/Settings/Billing payment failed banner",
  component: BillingPaymentFailedBanner,
  args: {
    failedAt: "2026-10-20T10:00:00Z",
    accessEndsAt: "2026-10-27T10:00:00Z",
    canManageBilling: true,
    pending: false,
    onUpdatePaymentMethod: () => console.log("Open Polar payment settings"),
  },
  decorators: [
    (Story) => (
      <div className="max-w-3xl p-4">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof BillingPaymentFailedBanner>;

export default meta;
type Story = StoryObj<typeof meta>;

export const DuringGrace: Story = {};
export const PaidAccessStopped: Story = { args: { accessEndsAt: undefined } };
export const UnknownFailureDate: Story = { args: { failedAt: undefined, accessEndsAt: undefined } };
export const OpeningPaymentSettings: Story = { args: { pending: true } };
export const WithoutBillingPermission: Story = { args: { canManageBilling: false } };
