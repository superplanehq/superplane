import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, waitFor, within } from "storybook/test";
import { BillingPlansTable } from "./BillingPlansTable";
import type { BillingPlanListItem } from "./billingPlansApi";

const systemReason = "System plans cannot be deleted.";
const polarReason = "This plan is linked to a Polar product and cannot be deleted.";
const assignedReason = "One or more organizations use this plan. Assign them another plan before deleting it.";

const basePlan: BillingPlanListItem = {
  id: "plan-spring",
  key: "spring_promo",
  name: "Spring promo",
  category: "paid",
  visibility: "private",
  require_payment_method: false,
  price_cents: 2900,
  per_seat: false,
  polar_product_id: null,
  seat_limit: 5,
  parallel_task_limit: 2,
  credit_amount_cents: 0,
  credit_frequency: "none",
  credit_expiration: "never",
  credit_validity_days: null,
  trial_credit_cents: 0,
  trial_length_days: 14,
  markup_bps: null,
  features: [],
  delete_disabled_reason: null,
  assigned: false,
};

const plans: BillingPlanListItem[] = [
  basePlan,
  {
    ...basePlan,
    id: "plan-free",
    key: "none",
    name: "Free",
    category: "none",
    visibility: "public",
    price_cents: 0,
    delete_disabled_reason: systemReason,
  },
  {
    ...basePlan,
    id: "plan-polar",
    key: "team_monthly",
    name: "Team monthly",
    category: "paid",
    visibility: "public",
    price_cents: 4900,
    per_seat: true,
    polar_product_id: "prod_team_monthly",
    delete_disabled_reason: polarReason,
  },
  {
    ...basePlan,
    id: "plan-assigned",
    key: "pilot",
    name: "Pilot",
    delete_disabled_reason: assignedReason,
    assigned: true,
  },
];

function InteractiveBillingPlansTable(args: { plans: BillingPlanListItem[] }) {
  const [rows, setRows] = useState(args.plans);
  return (
    <BillingPlansTable
      plans={rows}
      onEdit={(plan) => console.log("Edit plan", plan.key)}
      onDelete={(plan) => {
        console.log("Delete plan", plan.key);
        setRows((current) => current.filter((row) => row.id !== plan.id));
      }}
    />
  );
}

const meta = {
  title: "Admin/BillingPlansTable",
  component: BillingPlansTable,
  parameters: { layout: "padded" },
  args: { plans },
  render: (args) => <InteractiveBillingPlansTable plans={args.plans} />,
} satisfies Meta<typeof BillingPlansTable>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Mixed: Story = {};

export const HoverBlockedDelete: Story = {
  play: async () => {
    const trigger = await within(document.body).findByRole("group", { name: "Delete Team monthly is unavailable" });
    await userEvent.hover(trigger);
    await expect(await within(document.body).findByRole("tooltip")).toHaveTextContent(polarReason);
  },
};

export const KeyboardFocusBlockedDelete: Story = {
  play: async () => {
    const canvas = within(document.body);
    (await canvas.findByRole("button", { name: "Edit Pilot" })).focus();
    await userEvent.tab();
    await expect(canvas.getByRole("group", { name: "Delete Pilot is unavailable" })).toHaveFocus();
    await expect(await canvas.findByRole("tooltip")).toHaveTextContent(assignedReason);
    await expect(canvas.getByRole("button", { name: "Delete Pilot" })).toBeDisabled();
  },
};

export const DeleteEnabledPlan: Story = {
  play: async () => {
    const canvas = within(document.body);
    await userEvent.click(await canvas.findByRole("button", { name: "Delete Spring promo" }));
    const dialog = await canvas.findByRole("dialog", { name: "Delete plan?" });
    await expect(dialog).toHaveTextContent("Delete “Spring promo” (spring_promo)?");
    await expect(canvas.getByText("Spring promo")).toBeInTheDocument();
    await userEvent.click(within(dialog).getByRole("button", { name: "Delete plan" }));
    await waitFor(() => expect(canvas.queryByText("Spring promo")).not.toBeInTheDocument());
    await expect(canvas.getByRole("button", { name: "Delete Pilot" })).toBeDisabled();
  },
};

export const CancelDeleteEnabledPlan: Story = {
  play: async () => {
    const canvas = within(document.body);
    const deleteButton = await canvas.findByRole("button", { name: "Delete Spring promo" });
    await userEvent.click(deleteButton);
    const dialog = await canvas.findByRole("dialog", { name: "Delete plan?" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Keep plan" }));
    await waitFor(() => expect(canvas.queryByRole("dialog")).not.toBeInTheDocument());
    await expect(canvas.getByText("Spring promo")).toBeInTheDocument();
    await expect(deleteButton).toHaveFocus();
  },
};
