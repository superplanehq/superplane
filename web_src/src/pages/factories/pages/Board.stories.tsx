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
import { fiveStepLineFactoriesFixture } from "../__fixtures__/lineMetricsFactoriesFixture";
import {
  githubAndSentryIntakeFactoriesFixture,
  noIntakeFactoriesFixture,
  severalIntakeFactoriesFixture,
} from "../__fixtures__/backlogIntakeItemFixtures";
import {
  BUSINESS_ORGANIZATION_BILLING,
  LAPSED_ORGANIZATION_BILLING,
  LAPSED_TOPUP_USAGE_REPORT,
  LOW_CREDIT_USAGE_REPORT,
  SPENT_CREDIT_USAGE_REPORT,
} from "../__fixtures__/usageReportFixtures";
import { SENTRY_SETUP_INTEGRATIONS, JIRA_SETUP_INTEGRATIONS } from "../__fixtures__/setupStoryFixtures";
import {
  columnAutomationsEmptyPhaseFixture,
  columnAutomationsFixture,
  columnAutomationsNeedsRepairFixture,
  columnAutomationsSeveralIntakesFixture,
} from "../__fixtures__/columnAutomationsFixture";
import { boardConfidenceFixture } from "../__fixtures__/boardConfidenceFixture";
import { LinesPage } from "./LinesPage";

/**
 * Board is the workspace home: phase columns fill the pane. Cards open
 * the work-order popup.
 */
const meta = {
  title: "Factories/Pages/Board",
  component: LinesPage,
  parameters: {
    layout: "fullscreen",
    docs: { description: { component: "The app board with fixture API responses and shipped UI defaults." } },
  },
} satisfies Meta<typeof LinesPage>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Populated: Story = {
  name: "Board",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={boardConfidenceFixture}
      />
    );
  },
};

export const AcmeOnboardingEmpty: Story = {
  name: "Acme onboarding — empty board",
  render: () => (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}`}
      factoriesFixture={boardConfidenceFixture}
      appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP, ACME_ONBOARDING_FACTORY_ID)}
    />
  ),
};

export const AcmeOnboardingIntake: Story = {
  name: "Acme onboarding — intake settings",
  render: () => (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`}
      factoriesFixture={boardConfidenceFixture}
      appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP, ACME_ONBOARDING_FACTORY_ID)}
    />
  ),
};

export const LineBoardIntakeSettings: Story = {
  name: "Board — intake settings",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`}
        factoriesFixture={boardConfidenceFixture}
      />
    );
  },
};

export const LineBoardIntakeAutomation: Story = {
  name: "Board — intake automation",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}&settings=automation`}
        factoriesFixture={boardConfidenceFixture}
        appFixture={refundLineCanvasFixture(GITHUB_ISSUES_INTAKE_APP)}
      />
    );
  },
};

export const LineBoardGithubAndSentry: Story = {
  name: "Board — GitHub and Sentry listeners",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={githubAndSentryIntakeFactoriesFixture}
      />
    );
  },
};

export const SentryIntakeSetup: Story = {
  name: "Sentry intake setup — connection and project steps",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}/setup/sentry`}
        factoriesFixture={githubAndSentryIntakeFactoriesFixture}
        orgIntegrations={SENTRY_SETUP_INTEGRATIONS}
      />
    );
  },
};

export const JiraIntakeSetup: Story = {
  name: "Jira intake setup — connection and project steps",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}/setup/jira`}
        factoriesFixture={githubAndSentryIntakeFactoriesFixture}
        orgIntegrations={JIRA_SETUP_INTEGRATIONS}
      />
    );
  },
};

export const LineBoardSeveralIntakes: Story = {
  name: "Board — several intakes",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={severalIntakeFactoriesFixture}
      />
    );
  },
};

export const LineBoardNoIntakes: Story = {
  name: "Board — no intakes",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={noIntakeFactoriesFixture}
      />
    );
  },
};

export const LineBoardColumnAutomations: Story = {
  name: "Board — column automations",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardBacklogAutomationsOpen: Story = {
  name: "Board — Backlog automations open",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardVerifyAutomationsOpen: Story = {
  name: "Board — Verify automations open",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=verify`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardDoneAutomationsOpen: Story = {
  name: "Board — Done automations open",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=done`}
        factoriesFixture={columnAutomationsFixture}
      />
    );
  },
};

export const LineBoardAutomationView: Story = {
  name: "Board — automation view popup",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automationView=app-refund-implementer`}
        factoriesFixture={columnAutomationsFixture}
        appFixture={refundLineCanvasFixture()}
      />
    );
  },
};

export const LineBoardSeveralBacklogAutomations: Story = {
  name: "Board — several backlog automations",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={columnAutomationsSeveralIntakesFixture}
      />
    );
  },
};

export const LineBoardAutomationsNeedsRepair: Story = {
  name: "Board — needs repair",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=backlog`}
        factoriesFixture={columnAutomationsNeedsRepairFixture}
      />
    );
  },
};

export const LineBoardEmptyPhaseAutomations: Story = {
  name: "Board — empty phase automations",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}?automations=phase-3`}
        factoriesFixture={columnAutomationsEmptyPhaseFixture}
      />
    );
  },
};

export const LineDetailFivePhases: Story = {
  name: "Board — five phases",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={fiveStepLineFactoriesFixture}
      />
    );
  },
};

/** Welcome credit is empty. The board shows the trial chip next to the title. */
export const LineBoardHostedCreditEmpty: Story = {
  name: "Board — hosted credit empty",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={{ ...boardConfidenceFixture, organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT }}
      />
    );
  },
};

/** Purchased hosted credit remains at or below $20. The amber chip shows the balance next to the title. */
export const LineBoardHostedCreditLow: Story = {
  name: "Board — hosted credit low",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={{
          ...boardConfidenceFixture,
          organizationWorkspaceUsage: LOW_CREDIT_USAGE_REPORT,
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
        }}
      />
    );
  },
};

/** The organization has no plan. The amber chip sits next to the line title. */
export const LineBoardNoPlan: Story = {
  name: "Board — no plan",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        mode="app"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={{
          ...boardConfidenceFixture,
          organizationWorkspaceUsage: LAPSED_TOPUP_USAGE_REPORT,
          organizationBilling: LAPSED_ORGANIZATION_BILLING,
        }}
      />
    );
  },
};
