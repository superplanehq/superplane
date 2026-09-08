import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import type { ColumnAutomation } from "../lib/columnAutomations";
import { ColumnAutomationsHeaderSlot } from "./ColumnAutomationsIndicator";
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
    automation: INTAKE,
    onAction: () => undefined,
    defaultOpen: true,
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="relative min-h-[320px] bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof ColumnAutomationsPopup>;

export default meta;

type Story = StoryObj<typeof meta>;

export const InfoOpen: Story = {
  name: "Info open",
};

export const PhaseEditActions: Story = {
  name: "Phase edit actions",
  args: {
    automation: ANALYSIS,
    showEditAgent: true,
    showEditAutomation: true,
  },
};

export const NeedsRepair: Story = {
  name: "Needs repair",
  args: {
    automation: REPAIR,
  },
};

export const DisabledIcon: Story = {
  name: "Disabled icon",
  args: {
    automation: DISABLED,
  },
};

export const HeaderIcons: Story = {
  name: "Header icons",
  args: {
    defaultOpen: false,
  },
  render: () => (
    <ColumnAutomationsHeaderSlot
      title="Backlog"
      automations={[INTAKE, ANALYSIS, REPAIR]}
      onRowAction={() => undefined}
      testId="story-column-automations"
    />
  ),
};
