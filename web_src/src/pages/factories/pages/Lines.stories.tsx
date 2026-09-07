import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { refundLineCanvasFixture } from "../__fixtures__/factoryOwnedCanvasFixture";
import {
  ACME_ONBOARDING_FACTORY_ID,
  ACME_ONBOARDING_FACTORY_KEY,
  ACME_ONBOARDING_LINE_ID,
  GITHUB_ISSUES_INTAKE_APP,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { fiveStepLineFactoriesFixture, lineMetricsFactoriesFixture } from "../__fixtures__/lineMetricsFactoriesFixture";
import {
  githubAndSentryIntakeFactoriesFixture,
  noIntakeFactoriesFixture,
  severalIntakeFactoriesFixture,
} from "../__fixtures__/backlogIntakeItemFixtures";
import { LinesPage } from "./LinesPage";

/**
 * Line board is the workspace home: phase columns fill the pane. Cards open
 * the work-order popup. The backlog plus menu includes Create with an Agent.
 */
const meta = {
  title: "Factories/Pages/Lines",
  component: LinesPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof LinesPage>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Populated: Story = {
  name: "Line board",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />
    );
  },
};

export const AcmeOnboardingEmpty: Story = {
  name: "Acme onboarding — empty board",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}`}
      factoriesFixture={lineMetricsFactoriesFixture}
      appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP, ACME_ONBOARDING_FACTORY_ID)}
    />
  ),
};

export const AcmeOnboardingIntake: Story = {
  name: "Acme onboarding — intake settings",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`}
      factoriesFixture={lineMetricsFactoriesFixture}
      appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP, ACME_ONBOARDING_FACTORY_ID)}
    />
  ),
};

export const LineBoardIntakeSettings: Story = {
  name: "Line board — intake settings",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />
    );
  },
};

export const LineBoardIntakeAutomation: Story = {
  name: "Line board — intake automation",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}&settings=automation`}
        factoriesFixture={lineMetricsFactoriesFixture}
        appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP)}
      />
    );
  },
};

export const LineBoardIntakeRuns: Story = {
  name: "Line board — intake runs",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}&settings=runs`}
        factoriesFixture={lineMetricsFactoriesFixture}
        appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP)}
      />
    );
  },
};

export const LineBoardGithubAndSentry: Story = {
  name: "Line board — GitHub and Sentry listeners",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={githubAndSentryIntakeFactoriesFixture}
      />
    );
  },
};

export const LineBoardSeveralIntakes: Story = {
  name: "Line board — several intakes",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={severalIntakeFactoriesFixture}
      />
    );
  },
};

export const LineBoardNoIntakes: Story = {
  name: "Line board — no intakes",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={noIntakeFactoriesFixture}
      />
    );
  },
};

export const LineDetailFivePhases: Story = {
  name: "Line detail — five phases",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={fiveStepLineFactoriesFixture}
      />
    );
  },
};
