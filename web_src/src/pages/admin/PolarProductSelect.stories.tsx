import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { useState } from "react";
import { PolarProductSelect } from "./PolarProductSelect";

function InteractiveProductSelect(args: React.ComponentProps<typeof PolarProductSelect>) {
  const [value, setValue] = useState(args.value);
  return (
    <div className="max-w-md">
      <PolarProductSelect
        {...args}
        value={value}
        onChange={(next) => {
          console.log("Select Polar product", next);
          setValue(next);
        }}
      />
    </div>
  );
}

const meta = {
  title: "Admin/PolarProductSelect",
  component: PolarProductSelect,
  args: { kind: "subscription", value: "", onChange: (value) => console.log("Select Polar product", value) },
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/polar/products", () =>
          HttpResponse.json([
            { id: "prod_business", name: "Business monthly" },
            { id: "prod_team", name: "Team per seat" },
          ]),
        ),
      ],
    },
  },
  render: (args) => <InteractiveProductSelect {...args} />,
} satisfies Meta<typeof PolarProductSelect>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Subscriptions: Story = {};

export const LinkedSubscription: Story = {
  args: {
    value: "prod_business",
    disabled: true,
    description: "This plan is linked to Polar. Create a new plan to change the product or price.",
  },
};
