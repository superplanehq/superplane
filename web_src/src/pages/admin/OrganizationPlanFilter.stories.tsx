import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { useState } from "react";
import { userEvent, within } from "storybook/test";

import { OrganizationPlanFilter } from "./OrganizationPlanFilter";

const plans = [
  { key: "none", name: "None" },
  { key: "trial", name: "Trial" },
  { key: "business", name: "Business" },
  { key: "custom_team", name: "Custom Team" },
];

function InteractivePlanFilter(args: React.ComponentProps<typeof OrganizationPlanFilter>) {
  const [planKey, setPlanKey] = useState(args.planKey);
  return (
    <OrganizationPlanFilter
      {...args}
      planKey={planKey}
      onPlanChange={(key) => {
        console.log("Filter organizations by plan", key);
        setPlanKey(key);
      }}
    />
  );
}

const meta = {
  title: "Admin/OrganizationPlanFilter",
  component: OrganizationPlanFilter,
  parameters: {
    layout: "padded",
    msw: {
      handlers: [http.get("/admin/api/billing-plans", () => HttpResponse.json(plans))],
    },
  },
  args: { planKey: "", onPlanChange: (key) => console.log("Filter organizations by plan", key) },
  render: (args) => <InteractivePlanFilter {...args} />,
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole("combobox", { name: "Filter by plan" }));
  },
} satisfies Meta<typeof OrganizationPlanFilter>;

export default meta;
type Story = StoryObj<typeof meta>;

export const AllPlans: Story = {};

export const SelectedCustomPlan: Story = {
  args: { planKey: "custom_team" },
};

export const Loading: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/billing-plans", async () => {
          await delay("infinite");
          return HttpResponse.json([]);
        }),
      ],
    },
  },
};

export const LoadError: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/billing-plans", () => new HttpResponse(null, { status: 503 }), { once: true }),
        http.get("/admin/api/billing-plans", () => HttpResponse.json(plans)),
      ],
    },
  },
};

export const EmptyCatalog: Story = {
  parameters: {
    msw: {
      handlers: [http.get("/admin/api/billing-plans", () => HttpResponse.json([]))],
    },
  },
};
