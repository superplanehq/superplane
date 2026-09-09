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
import { LOW_CREDIT_USAGE_REPORT, SPENT_CREDIT_USAGE_REPORT } from "../__fixtures__/usageReportFixtures";
import {
  columnAutomationsEmptyPhaseFixture,
  columnAutomationsFixture,
  columnAutomationsNeedsRepairFixture,
  columnAutomationsSeveralIntakesFixture,
} from "../__fixtures__/columnAutomationsFixture";
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

export const LineBoardGithubAndSentry: Story = {
  name: "Line board — GitHub and Sentry listeners",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
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
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
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
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={noIntakeFactoriesFixture}
      />
    );
  },
};

export const LineBoardIntakeBannersLegacy: Story = {
  name: "Line board — intake banners (legacy)",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={githubAndSentryIntakeFactoriesFixture}
        previewFlags={{ addIntakeControl: true, columnAutomations: false }}
      />
    );
  },
};

export const LineBoardColumnAutomations: Story = {
  name: "Line board — column automations",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardBacklogAutomationsOpen: Story = {
  name: "Line board — Backlog automations open",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardVerifyAutomationsOpen: Story = {
  name: "Line board — Verify automations open",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=verify`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardDoneAutomationsOpen: Story = {
  name: "Line board — Done automations open",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=done`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardAutomationView: Story = {
  name: "Line board — automation view popup",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automationView=app-refund-implementer`}
        factoriesFixture={columnAutomationsFixture}
        appFixture={refundLineCanvasFixture()}
      />
    );
  },
};

export const LineBoardAddAutomationPicker: Story = {
  name: "Line board — add automation picker",
  parameters: {
    docs: {
      description: {
        story: "Open the column menu and press Add automation. Taken catalog entries stay disabled.",
      },
    },
  },
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardSeveralBacklogAutomations: Story = {
  name: "Line board — several backlog automations",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={columnAutomationsSeveralIntakesFixture}
      />
    );
  },
};

export const LineBoardAutomationsNeedsRepair: Story = {
  name: "Line board — needs repair",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={columnAutomationsNeedsRepairFixture}
      />
    );
  },
};

export const LineBoardEmptyPhaseAutomations: Story = {
  name: "Line board — empty phase automations",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=phase-3`}
        factoriesFixture={columnAutomationsEmptyPhaseFixture}
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

/** Welcome credit is empty. The board shows the trial-empty banner in the header. */
export const LineBoardHostedCreditEmpty: Story = {
  name: "Line board — hosted credit empty",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={{ ...lineMetricsFactoriesFixture, organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT }}
      />
    );
  },
};

/** Purchased hosted credit remains at or below $20. The board shows a low-credit warning. */
export const LineBoardHostedCreditLow: Story = {
  name: "Line board — hosted credit low",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={{ ...lineMetricsFactoriesFixture, organizationWorkspaceUsage: LOW_CREDIT_USAGE_REPORT }}
      />
    );
  },
};
