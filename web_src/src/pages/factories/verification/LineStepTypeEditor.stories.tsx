import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import { LineStepTypeEditor } from "./LineStepTypeEditor";
import { DEFAULT_SUITE_CHECKS, SUITE_OPTIONS } from "./__fixtures__/verificationFixtures";

const APPS = [
  { id: "app-build", name: "Build" },
  { id: "app-review", name: "Request review" },
];

/**
 * Design variant of the line step editor with a step type choice. `Verify`
 * shows the suite picker, its rule set, and a blocking toggle per check.
 */
const meta = {
  title: "Factories/Verification/LineStepTypeEditor",
  component: LineStepTypeEditor,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[420px] max-w-3xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof LineStepTypeEditor>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A verify step with the default quality suite selected. */
export const VerifyStep: Story = {
  args: {
    index: 1,
    value: {
      name: "verify",
      type: "verify",
      suiteId: "suite-default-quality",
      ruleSetName: "Production",
      checks: DEFAULT_SUITE_CHECKS,
    },
    suites: SUITE_OPTIONS,
    apps: APPS,
    onChange: (step) => console.log("step changed", step),
  },
};

/** The existing run-app step shape, for comparison with the verify step. */
export const RunAppStep: Story = {
  args: {
    index: 0,
    value: {
      name: "implement",
      type: "runApp",
      suiteId: "",
      ruleSetName: "",
      checks: [],
    },
    suites: SUITE_OPTIONS,
    apps: APPS,
    onChange: (step) => console.log("step changed", step),
  },
};

/** A verify step before a suite is selected. */
export const VerifyStepWithoutSuite: Story = {
  args: {
    index: 1,
    value: {
      name: "verify",
      type: "verify",
      suiteId: "",
      ruleSetName: "",
      checks: [],
    },
    suites: SUITE_OPTIONS,
    apps: APPS,
    onChange: (step) => console.log("step changed", step),
  },
};
