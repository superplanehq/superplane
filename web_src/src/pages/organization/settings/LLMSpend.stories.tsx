import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "@/pages/factories/__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture } from "@/pages/factories/__fixtures__/factoryPageResponses";
import { EMPTY_USAGE_REPORT } from "@/pages/factories/__fixtures__/usageReportFixtures";
import { OrganizationSettings } from "./index";

/**
 * Organization Settings → LLM spend. Factory-linked token and USD totals
 * for the last 30 days, separate from plan-limits Usage.
 */
const meta = {
  title: "Organization/Pages/Settings",
  component: OrganizationSettings,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof OrganizationSettings>;

export default meta;

type Story = StoryObj<typeof meta>;

export const LLMSpendPopulated: Story = {
  name: "LLM spend",
  render: () => <FactoriesHarness pathSuffix="settings/llm-spend" factoriesFixture={defaultFactoriesFixture} />,
};

export const LLMSpendEmpty: Story = {
  name: "LLM spend (empty)",
  render: () => (
    <FactoriesHarness
      pathSuffix="settings/llm-spend"
      factoriesFixture={{ ...defaultFactoriesFixture, organizationLlmSpend: EMPTY_USAGE_REPORT }}
    />
  ),
};
