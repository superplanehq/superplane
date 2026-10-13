import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import { RuleSetEditor } from "./RuleSetEditor";
import { EMPTY_RULE_SET, PRODUCTION_RULE_SET } from "./__fixtures__/verificationFixtures";

/**
 * Editor for an org-scoped rule set: rules grouped by domain with severity
 * and enforcement controls, and a YAML preview of the same data.
 */
const meta = {
  title: "Factories/Verification/RuleSetEditor",
  component: RuleSetEditor,
  parameters: { layout: "fullscreen" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-screen bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof RuleSetEditor>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The default production rule set with rules in every domain. */
export const Populated: Story = {
  args: {
    value: PRODUCTION_RULE_SET,
    onChange: (ruleSet) => console.log("rule set changed", ruleSet),
  },
};

/** Empty state for a new rule set. */
export const Empty: Story = {
  args: {
    value: EMPTY_RULE_SET,
    onChange: (ruleSet) => console.log("rule set changed", ruleSet),
  },
};
