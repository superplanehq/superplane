import type { Meta, StoryObj } from "@storybook/react-vite";

import { FleetManagerStep } from "./FleetManagerStep";

const meta = {
  title: "Auth/Owner setup/Fleet Manager",
  component: FleetManagerStep,
  parameters: { layout: "centered" },
  args: {
    onContinue: () => {
      // eslint-disable-next-line no-console
      console.log("Continue");
    },
    onSkip: () => {
      // eslint-disable-next-line no-console
      console.log("Skip");
    },
  },
} satisfies Meta<typeof FleetManagerStep>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
