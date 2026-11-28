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

  it.each([
    ["custom-qa", ["qa-manual"], undefined, true],
    ["phase-0", ["qa-manual"], undefined, false],
    ["custom-qa", undefined, undefined, false],
    ["custom-qa", ["other-app"], undefined, false],
    ["custom-qa", ["qa-manual"], "qa-manual", false],
    ["custom-qa", ["qa-manual"], "other-app", true],
  ] as const)(
    "checks custom-stage manual membership and queue: %s, %s, %s",
    (currentStageKey, manualStageAppIds, queuedStageAppId, eligible) => {
      const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER, { demoArtifacts: false });
      fixture.currentStageKey = currentStageKey;
      fixture.manualStageAppIds = manualStageAppIds ? [...manualStageAppIds] : undefined;
      fixture.queuedStageAppId = queuedStageAppId;
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
      if (eligible) expect(action).toBeEnabled();
      else expect(action).toBeNull();
    },
  );

  it.each(["failed", "cancelled", "passed", "running", "pending", "waiting"] as const)(
    "uses the latest matching non-history phase for custom-stage retries: %s",
    (status) => {
      const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER, { demoArtifacts: false });
      fixture.currentStageKey = "custom-qa";
      fixture.manualStageAppIds = ["qa-manual"];
      fixture.stageColumns = [{ id: "custom-qa", title: "QA" }, ...(fixture.stageColumns ?? [])];
      const phase = {
        ...fixture.phases[0],
        appId: "qa-manual",
        columnKey: "custom-qa",
        historyRun: false,
        componentName: "Update Docs",
        artifacts: [],
        stream: [],
        canvasSteps: [],
      };
      const now = Date.now();
      fixture.phases = [
        {
          ...phase,
          id: "earlier-match",
          status: status === "failed" || status === "cancelled" ? "passed" : "failed",
          startedAt: new Date(now).toISOString(),
        },
        { ...phase, id: "latest-match", status, startedAt: new Date(now - 1000).toISOString() },
        { ...phase, id: "history", status: "passed", historyRun: true },
        { ...phase, id: "other-app", status: "passed", appId: "other-app", componentName: "Other automation" },
        { ...phase, id: "other-column", status: "passed", columnKey: "verify" },
      ];
      const originalPhases = [...fixture.phases];
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
      if (status === "failed" || status === "cancelled") expect(action).toBeEnabled();
      else expect(action).toBeNull();
      expect(fixture.phases).toEqual(originalPhases);
    },
  );

  it.each([
    [0, false, "running", true],
    [0, false, "waiting", true],
    [1, false, "running", false],
    [0, true, "running", false],
    [0, false, "passed", false],
  ] as const)(
    "checks Implement manual eligibility: step %s, Verify %s, line %s",
    (currentStepIndex, canRunVerifyCustomAutomation, lineStatus, eligible) => {
      const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER, { demoArtifacts: false });
      Object.assign(fixture, { currentStepIndex, canRunVerifyCustomAutomation, lineStatus });
      renderConsole(fixture, {
        organizationId: "org-1",
        factoryId: "factory-1",
        orderId: BOARD_IMPLEMENT_NOTIFY_ORDER.id,
        canStopRun: true,
        manualCustomAutomations: [{ id: "implement-custom", name: "Inspect implementation", columnKey: "implement" }],
      });
      const action = within(screen.getByTestId("redesign-console-column-implement")).queryByRole("button", {
        name: /Run/,
      });
      if (eligible) expect(action).toBeEnabled();
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
