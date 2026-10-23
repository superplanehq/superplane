import type { Meta, StoryObj } from "@storybook/react-vite";
import { BillingSeatCheckout } from "./BillingSeatCheckout";

const meta = {
  title: "Factories/Settings/Billing seat checkout",
  component: BillingSeatCheckout,
  args: {
    plan: { key: "team", name: "Team", perSeat: true, priceCents: "2000", seatMaximum: 50 },
    occupied: 2,
    disabled: false,
    onSubscribe: (seats: number) => console.log("Subscribe to Team", { seats }),
  },
  decorators: [
    (Story) => (
      <div className="max-w-sm rounded border bg-background p-4">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof BillingSeatCheckout>;

export default meta;
type Story = StoryObj<typeof meta>;

export const TwoMembers: Story = {};
export const Pending: Story = { args: { disabled: true } };
export const Unlimited: Story = {
  args: { plan: { ...meta.args.plan, seatMaximum: undefined } },
};
export const BelowOccupiedMaximum: Story = {
  args: { plan: { ...meta.args.plan, seatMaximum: 1 } },
};
export const MissingSeatUsage: Story = { args: { occupied: undefined } };
