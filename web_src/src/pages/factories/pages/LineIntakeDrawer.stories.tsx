import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  GITHUB_ISSUES_INTAKE_APP,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { refundLineCanvasFixture } from "../__fixtures__/factoryOwnedCanvasFixture";
import { LineIntakeDrawer } from "./LineIntakeDrawer";

const meta = {
  title: "Factories/Pages/Intake tree",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const GitHubIssuesExpanded: Story = {
  name: "GitHub issues expanded",
  render: () => (
    <ComponentStoryShell className="flex h-svh bg-slate-300 p-0 dark:bg-slate-900">
      <LineIntakeDrawer initialSourceId="github-issues" onClose={() => undefined} />
    </ComponentStoryShell>
  ),
};

export const GitHubIssuesSettings: Story = {
  name: "GitHub issues settings",
  render: () => (
    <ComponentStoryShell className="flex h-svh bg-slate-300 p-0 dark:bg-slate-900">
      <LineIntakeDrawer initialSourceId="github-issues" initialSettingsOpen onClose={() => undefined} />
    </ComponentStoryShell>
  ),
};

export const GitHubIssuesAutomation: Story = {
  name: "GitHub issues automation",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&source=github-issues&settings=automation`}
        appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP)}
      />
    );
  },
};

export const GitHubIssuesRuns: Story = {
  name: "GitHub issues runs",
  render: () => (
    <ComponentStoryShell className="flex h-svh bg-slate-300 p-0 dark:bg-slate-900">
      <LineIntakeDrawer
        initialSourceId="github-issues"
        initialSettingsOpen
        initialSettingsTab="runs"
        onClose={() => undefined}
      />
    </ComponentStoryShell>
  ),
};
