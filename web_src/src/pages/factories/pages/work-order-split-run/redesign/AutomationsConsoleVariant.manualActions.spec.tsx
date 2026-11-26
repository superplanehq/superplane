import { renderConsole } from "./consoleTestSupport";
import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { DRAFT_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { CLOSED_FAILED_WORK_ORDER } from "../../../__fixtures__/factoryPageWorkOrders";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../../__fixtures__/lineMetricsBoardOrders";
import { splitRunFixtureForWorkOrder } from "../splitRunMocks";
import { backlogAnalysisRuns } from "../../../lib/backlogAnalysis";

const manualCustomAutomations = [
  { id: "backlog-custom", name: "Inspect task", columnKey: "backlog" as const },
  { id: "verify-custom", name: "Inspect pull request", columnKey: "verify" as const },
  { id: "done-custom", name: "Close the books", columnKey: "done" as const },
];

describe("AutomationsConsoleVariant manual actions", () => {
  it("keeps manual custom-stage runs in task history after the task leaves", () => {
    const order = {
      ...BOARD_IMPLEMENT_NOTIFY_ORDER,
      id: "wo-manual-stage-history",
      lineDispatches: BOARD_IMPLEMENT_NOTIFY_ORDER.lineDispatches?.map((dispatch) => ({
        ...dispatch,
        currentStageKey: "verify",
      })),
    };
    const runs = backlogAnalysisRuns("qa-manual", [
      {
        id: "manual-run",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        task: { id: order.id },
        createdAt: new Date().toISOString(),
      },
    ]);
    const fixture = splitRunFixtureForWorkOrder(order, {
      demoArtifacts: false,
      backlogCustomRuns: runs,
      columnApps: [{ id: "qa-manual", name: "Manual QA", columnKey: "custom-qa" }],
    });
    fixture.stageColumns = [
      { id: "backlog", title: "Backlog" },
      { id: "custom-qa", title: "QA" },
      { id: "verify", title: "Verify" },
    ];
    renderConsole(fixture, {
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: order.id,
      canStopRun: true,
      manualCustomAutomations: [{ id: "qa-manual", name: "Manual QA", columnKey: "custom-qa" }],
    });
    const column = within(screen.getByTestId("redesign-console-column-custom-qa"));
    expect(column.getByText("Manual QA")).toBeInTheDocument();
    expect(column.queryByRole("button", { name: /Run/ })).not.toBeInTheDocument();
  });
  it.each(["custom-qa", "phase-0"])("offers custom-stage Run only in its saved stage: %s", (currentStageKey) => {
    const order = {
      ...BOARD_IMPLEMENT_NOTIFY_ORDER,
      state: "STATE_OPEN" as const,
      pullRequests: [],
      lineDispatches: BOARD_IMPLEMENT_NOTIFY_ORDER.lineDispatches?.map((dispatch) => ({
        ...dispatch,
        currentStageKey,
      })),
    };
    const fixture = splitRunFixtureForWorkOrder(order, { demoArtifacts: false });
    fixture.stageColumns = [
      { id: "backlog", title: "Backlog" },
      { id: "custom-qa", title: "QA" },
      { id: "phase-0", title: "Implement" },
      { id: "verify", title: "Verify" },
      { id: "done", title: "Done" },
    ];
    renderConsole(fixture, {
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: order.id,
      canStopRun: true,
      manualCustomAutomations: [{ id: "qa-manual", name: "Manual QA", columnKey: "custom-qa" }],
    });
    if (currentStageKey === "custom-qa") {
      expect(screen.getByText("Manual QA")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /Run/ })).toBeEnabled();
    } else {
      expect(screen.queryByText("Manual QA")).not.toBeInTheDocument();
    }
  });
  it("offers only Backlog manual actions for a Backlog task", () => {
    renderConsole(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, { demoArtifacts: false }), {
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: DRAFT_WORK_ORDER.id,
      canStopRun: true,
      manualCustomAutomations,
    });
    expect(within(screen.getByTestId("redesign-console-column-backlog")).getByText("Inspect task")).toBeInTheDocument();
    expect(screen.queryByText("Inspect pull request")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Run/ })).toBeEnabled();
  });

  it.each([false, true])("offers Verify manual actions while PR feedback is active: %s", (activeFeedback) => {
    const order = {
      ...BOARD_IMPLEMENT_NOTIFY_ORDER,
      id: "wo-verify-manual",
      lineDispatches: BOARD_IMPLEMENT_NOTIFY_ORDER.lineDispatches?.map((dispatch) => ({
        ...dispatch,
        state: "STATE_FINISHED" as const,
        result: "RESULT_PASSED" as const,
      })),
      pullRequests: [
        {
          id: "pr-1",
          workOrderId: "wo-verify-manual",
          state: "STATE_OPEN" as const,
          url: "https://github.com/acme/app/pull/1",
        },
      ],
    };
    const fixture = splitRunFixtureForWorkOrder(order, {
      demoArtifacts: false,
      prFeedbackRuns: activeFeedback
        ? [
            {
              canvasId: "feedback",
              handlerName: "Address PR feedback",
              run: { id: "feedback-run", state: "STATE_STARTED" },
            },
          ]
        : [],
    });
    if (activeFeedback) expect(fixture.footer.attentionCard).not.toBe(true);
    renderConsole(fixture, {
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: order.id,
      canStopRun: true,
      manualCustomAutomations,
    });
    expect(
      within(screen.getByTestId("redesign-console-column-verify")).getByText("Inspect pull request"),
    ).toBeInTheDocument();
    expect(screen.queryByText("Inspect task")).not.toBeInTheDocument();
  });

  it.each(["RESULT_FAILED", "RESULT_PR_CLOSED"] as const)("offers Done manual actions for %s", (result) => {
    const order = { ...CLOSED_FAILED_WORK_ORDER, id: `wo-done-${result}`, result };
    renderConsole(splitRunFixtureForWorkOrder(order, { demoArtifacts: false }), {
      organizationId: "org-1",
      factoryId: "factory-1",
      orderId: order.id,
      canStopRun: true,
      manualCustomAutomations,
    });
    expect(within(screen.getByTestId("redesign-console-column-done")).getByText("Close the books")).toBeInTheDocument();
  });
});
