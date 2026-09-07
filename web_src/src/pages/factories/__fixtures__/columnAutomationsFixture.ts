import type { FactoriesFactoryIntake, FactoriesFactoryPrFeedbackHandler } from "@/api-client";

import {
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_PLAN_ID,
  type FactoriesFixture,
} from "./factoryPageResponses";
import { severalIntakeFactoriesFixture } from "./backlogIntakeItemFixtures";
import { lineMetricsFactoriesFixture } from "./lineMetricsFactoriesFixture";

export const PR_DISCUSSION_HANDLER_ID = "prfb-discussion";
export const PR_CHECKS_HANDLER_ID = "prfb-checks";

export const PR_DISCUSSION_HANDLER: FactoriesFactoryPrFeedbackHandler = {
  id: PR_DISCUSSION_HANDLER_ID,
  canvasId: "app-pr-discussion",
  name: "Address PR feedback",
  source: "SOURCE_PULL_REQUEST_DISCUSSION",
  healthy: true,
};

export const PR_CHECKS_HANDLER: FactoriesFactoryPrFeedbackHandler = {
  id: PR_CHECKS_HANDLER_ID,
  canvasId: "app-pr-checks",
  name: "Fix pull request checks",
  source: "SOURCE_PULL_REQUEST_CHECKS",
  healthy: true,
};

const UNHEALTHY_GITHUB_INTAKE: FactoriesFactoryIntake = {
  ...GITHUB_ISSUES_INTAKE,
  id: GITHUB_ISSUES_INTAKE_ID,
  healthy: false,
};

function withPRFeedbackHandlers(
  fixture: FactoriesFixture,
  handlers: FactoriesFactoryPrFeedbackHandler[],
): FactoriesFixture {
  return {
    ...fixture,
    prFeedbackHandlersByFactoryId: {
      ...fixture.prFeedbackHandlersByFactoryId,
      [PRIMARY_FACTORY_ID]: handlers,
    },
  };
}

/** Populated board: Backlog intake + analysis, Implement agent, Verify listeners, Done closure. */
export const columnAutomationsFixture: FactoriesFixture = withPRFeedbackHandlers(lineMetricsFactoriesFixture, [
  PR_DISCUSSION_HANDLER,
  PR_CHECKS_HANDLER,
]);

/** Same board with a GitHub intake that needs repair. */
export const columnAutomationsNeedsRepairFixture: FactoriesFixture = {
  ...columnAutomationsFixture,
  intakesByFactoryId: {
    ...columnAutomationsFixture.intakesByFactoryId,
    [PRIMARY_FACTORY_ID]: [UNHEALTHY_GITHUB_INTAKE],
  },
};

/** Backlog with several intakes, plus Verify listeners. */
export const columnAutomationsSeveralIntakesFixture: FactoriesFixture = withPRFeedbackHandlers(
  severalIntakeFactoriesFixture,
  [PR_DISCUSSION_HANDLER, PR_CHECKS_HANDLER],
);

/** Adds a Review phase with no canvas so the empty automations state is reachable. */
export const columnAutomationsEmptyPhaseFixture: FactoriesFixture = {
  ...columnAutomationsFixture,
  factories: columnAutomationsFixture.factories.map((factory) => {
    if (factory.id !== PRIMARY_FACTORY_ID) {
      return factory;
    }
    return {
      ...factory,
      lines: (factory.lines ?? []).map((line) => {
        if (line.id !== REFUND_LINE_PLAN_ID) {
          return line;
        }
        return {
          ...line,
          steps: [...(line.steps ?? []), { name: "Review", type: "runApp" }],
        };
      }),
    };
  }),
};
