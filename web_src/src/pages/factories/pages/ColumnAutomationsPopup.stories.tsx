import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationsPopup } from "./ColumnAutomationsPopup";

const INTAKE: ColumnAutomation = {
  id: "intake-github",
  kind: "intake",
  name: "GitHub issues",
  trigger: "On GitHub issue",
  action: "Create a task in Backlog",
  iconSrc: "",
  iconAlt: "GitHub",
  health: "healthy",
  runningCount: 0,
  catalogId: "github-issues",
  canvasId: "app-github",
};

const ANALYSIS: ColumnAutomation = {
  id: "analysis-1",
  kind: "analysis",
  name: "Task analysis",
  trigger: "On task in Backlog",
  action: "Score the task",
  iconSrc: "",
  iconAlt: "",
  health: "healthy",
  runningCount: 2,
  catalogId: "analysis",
  canvasId: "app-refund-backlog",
};

const REPAIR: ColumnAutomation = {
  ...INTAKE,
  id: "intake-sentry",
  name: "Sentry exceptions",
  trigger: "On Sentry exception",
  catalogId: "sentry-exceptions",
  health: "needs-repair",
};

const DISABLED: ColumnAutomation = {
  ...ANALYSIS,
  id: "analysis-disabled",
  health: "disabled",
  runningCount: 0,
};

const meta = {
  title: "Factories/Components/ColumnAutomationsPopup",
  component: ColumnAutomationsPopup,
  parameters: { layout: "centered" },
  args: {
    columnTitle: "Backlog",
    columnKey: "backlog",
    automations: [INTAKE, ANALYSIS],
    open: true,
    onClose: () => undefined,
    onAdd: () => undefined,
    onRowAction: () => undefined,
    trigger: (
      <button type="button" className="rounded-md border border-border px-2 py-1 text-sm">
        Automations
      </button>
    ),
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="relative min-h-[520px] bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof ColumnAutomationsPopup>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Populated: Story = {
  name: "Populated",
};

export const Empty: Story = {
  name: "Empty phase",
  args: {
    columnTitle: "Review",
    columnKey: "phase-1",
    automations: [],
  },
};

export const NeedsRepair: Story = {
  name: "Needs repair",
  args: {
    automations: [REPAIR, ANALYSIS],
  },
};

export const DisabledRow: Story = {
  name: "Disabled row",
  args: {
    automations: [INTAKE, DISABLED],
  },
};
