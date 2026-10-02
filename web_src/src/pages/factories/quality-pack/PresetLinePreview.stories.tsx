import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import { VERIFIED_DELIVERY_STEPS } from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { PresetLinePreview } from "./PresetLinePreview";

/**
 * Install-time preview of the Verified delivery preset line. The verify step
 * is expanded so the gate and its checks are visible before install.
 */
const meta = {
  title: "Factories/Quality Pack/PresetLinePreview",
  component: PresetLinePreview,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[560px] max-w-3xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof PresetLinePreview>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The four steps of the Verified delivery line, with the verify step expanded. */
export const VerifiedDelivery: Story = {
  args: {
    lineName: "verified-delivery",
    steps: VERIFIED_DELIVERY_STEPS,
    onInstall: () => console.log("install line"),
  },
};
