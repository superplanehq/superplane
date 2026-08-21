import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import { QUALITY_TEMPLATES } from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { QualityTemplateGallery } from "./QualityTemplateGallery";

/**
 * Gallery of the six installable quality templates from the code quality
 * pack. Each template is one canvas that runs standalone or as a check
 * inside a verification suite.
 */
const meta = {
  title: "Factories/Quality Pack/QualityTemplateGallery",
  component: QualityTemplateGallery,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[480px] max-w-5xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
  args: {
    onInstall: (templateId) => console.log("install template", templateId),
  },
} satisfies Meta<typeof QualityTemplateGallery>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The full pack; one template is already installed. */
export const Default: Story = {
  args: { templates: QUALITY_TEMPLATES },
};
