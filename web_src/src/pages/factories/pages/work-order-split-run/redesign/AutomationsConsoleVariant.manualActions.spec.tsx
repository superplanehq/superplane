import { renderConsole } from "./consoleTestSupport";
import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { DRAFT_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { CLOSED_FAILED_WORK_ORDER } from "../../../__fixtures__/factoryPageWorkOrders";
import { BOARD_IMPLEMENT_NOTIFY_ORDER } from "../../../__fixtures__/lineMetricsBoardOrders";
import { splitRunFixtureForWorkOrder } from "../splitRunMocks";

const manualCustomAutomations = [
  { id: "backlog-custom", name: "Inspect task", columnKey: "backlog" as const },
  { id: "verify-custom", name: "Inspect pull request", columnKey: "verify" as const },
  { id: "done-custom", name: "Close the books", columnKey: "done" as const },
];

describe("AutomationsConsoleVariant manual actions", () => {
  it.each([undefined, "RESULT_PASSED", "RESULT_FAILED"] as const)(
    "uses the current dispatch after a manual-stage restart: %s",
    (currentResult) => {
      const previous = BOARD_IMPLEMENT_NOTIFY_ORDER.lineDispatches![0];
      const execution = {
        ...previous.stepExecutions![0],
        stageKey: "custom-qa",
        step: "QA",
        run: { id: "old-run", appId: "qa-manual", appName: "Update Docs" },
      };
      const fixture = splitRunFixtureForWorkOrder(
        {
          ...BOARD_IMPLEMENT_NOTIFY_ORDER,
          id: "restarted-manual-task",
          lineDispatches: [
            { ...previous, id: "old-dispatch", state: "STATE_FINISHED", stepExecutions: [execution] },
            {
              ...previous,
              id: "current-dispatch",
              currentStageKey: "custom-qa",
              steps: [{ appId: "qa-manual", name: "QA", stageKey: "custom-qa", manualStart: true }],
              stepExecutions: currentResult
                ? [
                    {
                      ...execution,
                      id: "current-execution",
                      result: currentResult,
                      run: { ...execution.run, id: "current-run" },
                    },
                  ]
                : [],
            },
          ],
        },
        {
          demoArtifacts: false,
          lineStages: [{ key: "custom-qa", name: "QA", kind: "STAGE_KIND_PHASE" }],
        },
      );
      renderConsole(fixture, {
        organizationId: "org-1",
        factoryId: "factory-1",
        orderId: "restarted-manual-task",
        canStopRun: true,
        manualCustomAutomations: [{ id: "qa-manual", name: "Update Docs", columnKey: "custom-qa" }],
      });

      const action = within(screen.getByTestId("redesign-console-column-custom-qa")).queryByRole("button", {
        name: /Run/,
      });
      if (currentResult === "RESULT_PASSED") expect(action).toBeNull();
      else expect(action).toBeEnabled();
    },
  );

  it.each(["custom-qa", "phase-0"])(
    "offers unstarted custom work only in its matching stage: %s",
    (currentStageKey) => {
      const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER, { demoArtifacts: false });
      fixture.currentStageKey = currentStageKey;
      fixture.manualStageAppIds = ["qa-manual"];
      fixture.stageColumns = [{ id: "custom-qa", title: "QA" }, ...(fixture.stageColumns ?? [])];
      renderConsole(fixture, {
        organizationId: "org-1",
        factoryId: "factory-1",
        orderId: BOARD_IMPLEMENT_NOTIFY_ORDER.id,
        canStopRun: true,
        manualCustomAutomations: [{ id: "qa-manual", name: "Update Docs", columnKey: "custom-qa" }],
      });
      const action = within(screen.getByTestId("redesign-console-column-custom-qa")).queryByRole("button", {
        name: /Run/,
      });
      if (currentStageKey === "custom-qa") expect(action).toBeEnabled();
      else expect(action).toBeNull();
    },
  );

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
