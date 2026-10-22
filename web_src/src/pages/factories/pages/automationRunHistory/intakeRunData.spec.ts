import type { RunHistoryIntakeSource } from "./intakeRunData";
import { describe, expect, it } from "bun:test";
import { toAutomationRun, toRunActivities } from "./runHistoryData";
import type { CanvasesCanvasRun } from "@/api-client";

const received: CanvasesCanvasRun = {
  id: "received",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  rootEvent: {
    id: "event",
    data: {
      type: "github.issue",
      data: {
        action: "labeled",
        repository: { full_name: "team/project" },
        issue: { number: 42, title: "Fix checkout", html_url: "https://github.com/team/project/issues/42" },
      },
    },
  },
  executions: [
    { id: "filter", nodeId: "filter", componentName: "if", result: "RESULT_PASSED", outputChannels: ["false"] },
  ],
};
const map = (run: CanvasesCanvasRun) => toAutomationRun(run, [], Date.now(), "github-issues");

describe("Intake run evidence", () => {
  it("shows a received issue and a recorded filter rejection without a linked task", () => {
    const run = map(received);
    expect(run.source).toEqual({
      reference: "project#42",
      title: "Fix checkout",
      url: "https://github.com/team/project/issues/42",
    });
    expect(run.intake?.outcome).toBe("filtered");
    expect(run.status).toBe("Passed");
    expect(run.task).toBeUndefined();
  });
  it("does not call missing tasks or duplicate creation a filter rejection", () => {
    expect(map({ ...received, executions: [] }).intake?.outcome).toBe("unknown");
    expect(
      map({
        ...received,
        executions: [{ componentName: "createWorkOrder", result: "RESULT_PASSED", outputChannels: [] }],
      }).intake?.outcome,
    ).toBe("unknown");
    expect(
      map({
        ...received,
        executions: [{ componentName: "createWorkOrder", result: "RESULT_PASSED", outputChannels: ["default"] }],
      }).intake?.outcome,
    ).toBe("created");
  });
  it("uses a recorded duplicate decision to identify the existing task", () => {
    const run: CanvasesCanvasRun = {
      ...received,
      executions: [
        {
          id: "create",
          nodeId: "create",
          componentName: "createWorkOrder",
          result: "RESULT_PASSED",
          existingTaskId: "existing-task",
        },
      ],
    };
    const mapped = toAutomationRun(
      run,
      [{ id: "existing-task", key: "SP-42", title: "Fix checkout", sourceRunId: "original-run" }],
      Date.now(),
      "github-issues",
    );
    expect(mapped.intake?.outcome).toBe("existing");
    expect(mapped.task).toEqual({ id: "SP-42", title: "Fix checkout" });
    expect(map(run).intake?.outcome).toBe("existing");
    expect(map(run).task).toBeUndefined();
    expect(map({ ...run, result: "RESULT_FAILED" }).intake?.outcome).toBe("failed");
  });
  it("does not reuse an existing-task decision from a superseded or unfinished attempt", () => {
    const previous = {
      id: "first",
      nodeId: "create",
      componentName: "createWorkOrder",
      result: "RESULT_PASSED" as const,
      existingTaskId: "existing-task",
      createdAt: "2026-10-22T12:00:00Z",
    };
    expect(
      map({
        ...received,
        executions: [
          previous,
          { ...previous, id: "retry", existingTaskId: undefined, createdAt: "2026-10-22T12:01:00Z" },
        ],
      }).intake?.outcome,
    ).toBe("unknown");
    expect(map({ ...received, executions: [{ ...previous, result: "RESULT_UNKNOWN" }] }).intake?.outcome).toBe(
      "unknown",
    );
  });
  it("uses the last attempt for each filter and keeps failed runs failed", () => {
    const retry = {
      ...received.executions![0],
      id: "retry",
      createdAt: "2026-10-22T12:01:00Z",
      outputChannels: ["true"],
    };
    expect(map({ ...received, executions: [...received.executions!, retry] }).intake?.outcome).toBe("unknown");
    expect(map({ ...received, result: "RESULT_FAILED" }).intake?.outcome).toBe("failed");
  });
  it.each([
    {
      sourceId: "sentry-exceptions",
      data: {
        data: { issue: { shortId: "APP-12", title: "Checkout error", permalink: "https://sentry.io/issues/12" } },
      },
      key: "APP-12",
    },
    {
      sourceId: "linear-issues",
      data: {
        action: "create",
        type: "Issue",
        url: "https://linear.app/example/issue/ENG-42",
        data: { identifier: "ENG-42", title: "Checkout error" },
      },
      key: "ENG-42",
    },
    {
      sourceId: "jira-issues",
      data: {
        issue: { key: "PLAT-42", fields: { summary: "Checkout error" } },
        url: "https://example.atlassian.net/browse/PLAT-42",
      },
      key: "PLAT-42",
    },
    {
      sourceId: "dependabot-alerts",
      data: {
        alert: {
          number: 12,
          security_advisory: { summary: "Checkout error" },
          html_url: "https://github.com/team/project/security/dependabot/12",
        },
        repository: { full_name: "team/project" },
      },
      key: "project#12",
    },
    {
      sourceId: "productive-tasks",
      data: { data: { id: "563", attributes: { title: "Checkout error" } } },
      key: "#563",
    },
    {
      sourceId: "pagerduty-incidents",
      data: {
        incident: { id: "PN12", title: "Checkout error", html_url: "https://example.pagerduty.com/incidents/PN12" },
      },
      key: "PN12",
    },
    {
      sourceId: "datadog",
      data: { id: "4820", title: "Checkout error", link: "https://app.datadoghq.com/event/4820" },
      key: "4820",
    },
  ])("reads $sourceId source identity before a task exists", ({ sourceId, data, key }) => {
    const run = toAutomationRun(
      { ...received, rootEvent: { data: { type: "source.event", data } }, executions: [] },
      [],
      Date.now(),
      sourceId as RunHistoryIntakeSource,
    );
    expect(run.source?.reference).toBe(key);
    expect(run.source?.title).toBe("Checkout error");
    if (sourceId === "linear-issues") expect(run.source?.url).toBe("https://linear.app/example/issue/ENG-42");
    expect(run.task).toBeUndefined();
  });
  it("does not attach a different task that happens to share the incoming title", () => {
    const run = toAutomationRun(
      { ...received, rootEvent: { ...received.rootEvent, customName: "Fix checkout" } },
      [{ id: "other-task", title: "Fix checkout", sourceRunId: "another-run" }],
      Date.now(),
      "github-issues",
    );
    expect(run.task).toBeUndefined();
    expect(run.intake?.outcome).toBe("filtered");
  });
  it("keeps non-intake runs on execution status even when an if condition is false", () => {
    expect(toAutomationRun(received, [], Date.now()).intake).toBeUndefined();
  });
  it("opens source data and filter outputs first, but retains agent logs as the default", () => {
    const activities = toRunActivities(
      received,
      [
        { id: "filter", runId: "received", nodeId: "filter", outputs: { false: [{}] } },
        { id: "agent", runId: "received", nodeId: "agent", metadata: { runner_broker_task_id: "broker" } },
      ],
      [
        { id: "filter", component: "if" },
        { id: "agent", component: "runnerSuperPlane" },
      ],
    );
    expect(activities.find((a) => a.id === "event:event")?.outputsLabel).toBe("Event data");
    expect(activities.find((a) => a.id === "filter")?.defaultTab).toBe("outputs");
    expect(activities.find((a) => a.id === "agent")?.defaultTab).toBe("logs");
  });
  it("reports no proposal for a scheduled review without hiding its agent activity", () => {
    const run = toAutomationRun(
      {
        ...received,
        rootEvent: { data: { type: "schedule.tick", data: {} } },
        executions: [
          {
            id: "proposal",
            nodeId: "has-proposal",
            componentName: "if",
            result: "RESULT_PASSED",
            outputChannels: ["false"],
          },
        ],
      },
      [],
      Date.now(),
      "scheduled-intake",
    );
    expect(run.intake?.outcome).toBe("no-proposal");
  });
});
