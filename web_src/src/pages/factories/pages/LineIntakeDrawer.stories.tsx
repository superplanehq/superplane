import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_APP,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { refundLineCanvasFixture } from "../__fixtures__/factoryOwnedCanvasFixture";
import { LineIntakeDrawer } from "./LineIntakeDrawer";
import { GITHUB_ISSUES_ANALYZING_TICKETS, intakeSourcesFromFactoryIntakes } from "./lineIntakeModel";

const meta = {
  title: "Factories/Pages/Intake tree",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const CONFIGURED_INTAKES = intakeSourcesFromFactoryIntakes([
  GITHUB_ISSUES_INTAKE,
  { id: "intake-sentry", canvasId: "app-sentry-intake", name: "Sentry exceptions", source: "SOURCE_SENTRY_EXCEPTIONS" },
  {
    id: "intake-pagerduty",
    canvasId: "app-pagerduty-intake",
    name: "PagerDuty incidents",
    source: "SOURCE_PAGERDUTY_INCIDENTS",
  },
]);

const storyDrawerProps = { showAddIntakeControl: true, onClose: () => undefined } as const;

export const GitHubIssuesExpanded: Story = {
  name: "GitHub issues expanded",
  render: () => (
    <ComponentStoryShell className="flex h-svh bg-slate-300 p-0 dark:bg-slate-900">
      <LineIntakeDrawer
        {...storyDrawerProps}
        configuredSources={CONFIGURED_INTAKES}
        analyzingTickets={GITHUB_ISSUES_ANALYZING_TICKETS}
        initialIntakeId={GITHUB_ISSUES_INTAKE_ID}
      />
    </ComponentStoryShell>
  ),
};

export const TwoGitHubIntakes: Story = {
  name: "Two GitHub intakes",
  render: () => (
    <ComponentStoryShell className="flex h-svh bg-slate-300 p-0 dark:bg-slate-900">
      <LineIntakeDrawer
        {...storyDrawerProps}
        configuredSources={intakeSourcesFromFactoryIntakes([
          GITHUB_ISSUES_INTAKE,
          {
            id: "intake-github-triage",
            canvasId: "app-github-triage",
            name: "Triage issues",
            source: "SOURCE_GITHUB_ISSUES",
          },
          {
            id: "intake-github-broken",
            canvasId: "app-github-broken",
            name: "Security reports",
            source: "SOURCE_GITHUB_ISSUES",
            healthy: false,
          },
        ])}
        analyzingTickets={GITHUB_ISSUES_ANALYZING_TICKETS}
        initialIntakeId={GITHUB_ISSUES_INTAKE_ID}
      />
    </ComponentStoryShell>
  ),
};

export const GitHubIssuesSettings: Story = {
  name: "GitHub issues settings",
  render: () => (
    <ComponentStoryShell className="flex h-svh bg-slate-300 p-0 dark:bg-slate-900">
      <LineIntakeDrawer
        {...storyDrawerProps}
        configuredSources={CONFIGURED_INTAKES}
        initialIntakeId={GITHUB_ISSUES_INTAKE_ID}
        initialSettingsOpen
      />
    </ComponentStoryShell>
  ),
};

export const GitHubIssuesAutomation: Story = {
  name: "GitHub issues automation",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}&settings=automation`}
        appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP)}
      />
    );
  },
};

export const GitHubIssuesRuns: Story = {
  name: "GitHub issues runs",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}&settings=runs`}
        appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP)}
      />
    );
  },
};
