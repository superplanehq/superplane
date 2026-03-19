import type { Meta, StoryObj } from "@storybook/react-vite";
import { BrowserRouter } from "react-router-dom";
import { OnboardingWelcome } from "./OnboardingWelcome";

const meta: Meta<typeof OnboardingWelcome> = {
  title: "Pages/OnboardingWelcome",
  component: OnboardingWelcome,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="bg-slate-100 dark:bg-slate-900 min-h-screen">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    organizationId: "org-123",
    canCreateCanvases: true,
    permissionsLoading: false,
  },
};

export const PermissionsRestricted: Story = {
  args: {
    organizationId: "org-123",
    canCreateCanvases: false,
    permissionsLoading: false,
  },
};
