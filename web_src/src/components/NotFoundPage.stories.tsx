import type { Meta, StoryObj } from "@storybook/react-vite";
import { NotFoundPage } from "./NotFoundPage";

const meta: Meta<typeof NotFoundPage> = {
  title: "Components/NotFoundPage",
  component: NotFoundPage,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof NotFoundPage>;

export const FlightPlan: Story = {
  args: {
    description: "This plane has left the control plane.",
    actionLabel: "Return to the hangar",
    showFlightAnimation: true,
  },
};
