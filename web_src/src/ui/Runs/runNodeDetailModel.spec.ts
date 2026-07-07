import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import {
  buildRunInspectorNodeSections,
  buildTriggerTabData,
  getAdjacentRunNodeId,
  isRunNodeDetailTabAvailable,
  resolveRunNodeDetailTab,
} from "./runNodeDetailModel";

const allTabs = {
  hasDetailsSection: true,
  hasPayload: true,
  hasConfig: true,
};

describe("resolveRunNodeDetailTab", () => {
  it("returns the preferred tab when it is available", () => {
    expect(resolveRunNodeDetailTab("payload", allTabs)).toBe("payload");
  });

  it("falls back when the preferred tab is unavailable", () => {
    expect(
      resolveRunNodeDetailTab("payload", {
        hasDetailsSection: true,
        hasPayload: false,
        hasConfig: false,
      }),
    ).toBe("details");
  });

  it("checks tab availability", () => {
    expect(isRunNodeDetailTabAvailable("configuration", allTabs)).toBe(true);
    expect(
      isRunNodeDetailTabAvailable("payload", {
        hasDetailsSection: true,
        hasPayload: false,
        hasConfig: false,
      }),
    ).toBe(false);
  });
});

describe("getAdjacentRunNodeId", () => {
  const chain = ["trigger", "node-a", "node-b"];

  it("returns the previous node id", () => {
    expect(getAdjacentRunNodeId(chain, "node-b", "prev")).toBe("node-a");
  });

  it("returns the next node id", () => {
    expect(getAdjacentRunNodeId(chain, "node-a", "next")).toBe("node-b");
  });

  it("returns null at the ends of the chain", () => {
    expect(getAdjacentRunNodeId(chain, "trigger", "prev")).toBeNull();
    expect(getAdjacentRunNodeId(chain, "node-b", "next")).toBeNull();
  });
});

describe("buildTriggerTabData", () => {
  it("uses trigger mapper details for trigger nodes", () => {
    const run = buildGithubCommitStatusRun();
    const node = buildTriggerNode("github.onCommitStatus");

    const tabData = buildTriggerTabData(run, node);

    expect(tabData.details).toMatchObject({
      State: "success",
      Context: "ci/semaphoreci/pr: Operately - Build & Test",
      Description: "The build passed on Semaphore 2.0.",
      SHA: "43a3f0a1ac3ba8bc45e1b858c557094af374eddb",
      "Commit message": "Adjust skills",
      "Commit author": "Rockyy174",
      Branches: "-",
      Repository: "operately/operately",
      "Status creator": "shiroyasha",
      "Commit URL": "https://github.com/operately/operately/commit/43a3f0a1ac3ba8bc45e1b858c557094af374eddb",
      "Target URL": "https://operately.semaphoreci.com/workflows/5128f711",
    });
    expect(tabData.details).not.toHaveProperty("Channel");
    expect(tabData.details).not.toHaveProperty("Triggered at");
    expect(tabData.payload).toEqual(run.rootEvent?.data);
  });

  it("falls back to generic trigger details when the event has no mapper details", () => {
    const run: CanvasesCanvasRun = {
      rootEvent: {
        id: "event-1",
        nodeId: "trigger-1",
        channel: "default",
        customName: "Manual run",
        createdAt: new Date().toISOString(),
        data: {},
      },
    };

    const tabData = buildTriggerTabData(run, buildTriggerNode("unknown.trigger"));

    expect(tabData.details).toEqual({
      Channel: "default",
      Name: "Manual run",
      "Triggered at": run.rootEvent?.createdAt,
    });
  });
});

describe("buildRunInspectorNodeSections", () => {
  it("uses the root event payload as trigger output", () => {
    const run: CanvasesCanvasRun = {
      rootEvent: {
        id: "event-1",
        nodeId: "trigger",
        createdAt: new Date().toISOString(),
        data: { repository: "superplane" },
      },
    };

    const [triggerSection] = buildRunInspectorNodeSections({
      run,
      executions: [],
      workflowNodes: [
        {
          id: "trigger",
          name: "On Pull Request",
          type: "TYPE_TRIGGER",
          component: "github.onPullRequest",
        },
      ],
    });

    expect(triggerSection.isTrigger).toBe(true);
    expect(triggerSection.upstreamSections).toEqual([]);
    expect(triggerSection.outputSections).toEqual([
      {
        channel: "default",
        value: { repository: "superplane" },
        sizeKb: expect.any(String),
      },
    ]);
  });

  it("includes earlier inputs in topological order and marks the immediate previous step as primary", () => {
    const run: CanvasesCanvasRun = {
      rootEvent: {
        id: "event-1",
        nodeId: "trigger",
        createdAt: new Date().toISOString(),
        data: { trigger: true },
      },
    };

    const sections = buildRunInspectorNodeSections({
      run,
      executions: [
        {
          id: "execution-1",
          nodeId: "first-action",
          outputs: { default: [{ first: true }] },
          metadata: {},
        },
        {
          id: "execution-2",
          nodeId: "second-action",
          outputs: { default: [{ second: true }] },
          metadata: {},
        },
      ],
      workflowNodes: [
        { id: "trigger", name: "Trigger", type: "TYPE_TRIGGER", component: "github.onPullRequest" },
        { id: "first-action", name: "First Action", type: "TYPE_ACTION", component: "test.first" },
        { id: "second-action", name: "Second Action", type: "TYPE_ACTION", component: "test.second" },
      ],
    });

    const secondAction = sections.find((section) => section.nodeId === "second-action");

    expect(secondAction?.upstreamSections.map((section) => section.nodeName)).toEqual(["Trigger", "First Action"]);
    expect(secondAction?.upstreamSections.map((section) => section.output)).toEqual([
      { trigger: true },
      { first: true },
    ]);
    expect(secondAction?.primaryInputNodeId).toBe("first-action");
    expect(secondAction?.outputSections).toEqual([
      {
        channel: "default",
        value: { second: true },
        sizeKb: expect.any(String),
      },
    ]);
  });

  it("uses edges to include only accessible upstream nodes ordered by creation time", () => {
    const run: CanvasesCanvasRun = {
      rootEvent: {
        id: "event-1",
        nodeId: "trigger",
        createdAt: "2026-05-01T12:00:00Z",
        data: { trigger: true },
      },
    };

    const sections = buildRunInspectorNodeSections({
      run,
      executions: [
        {
          id: "execution-1",
          nodeId: "first-action",
          createdAt: "2026-05-01T12:00:03Z",
          outputs: { default: [{ first: true }] },
          metadata: {},
        },
        {
          id: "execution-2",
          nodeId: "independent-action",
          createdAt: "2026-05-01T12:00:01Z",
          outputs: { default: [{ independent: true }] },
          metadata: {},
        },
        {
          id: "execution-3",
          nodeId: "second-action",
          createdAt: "2026-05-01T12:00:04Z",
          outputs: { default: [{ second: true }] },
          metadata: {},
        },
      ],
      workflowNodes: [
        { id: "trigger", name: "Trigger", type: "TYPE_TRIGGER", component: "github.onPullRequest" },
        { id: "first-action", name: "First Action", type: "TYPE_ACTION", component: "test.first" },
        { id: "independent-action", name: "Independent Action", type: "TYPE_ACTION", component: "test.independent" },
        { id: "second-action", name: "Second Action", type: "TYPE_ACTION", component: "test.second" },
      ],
      workflowEdges: [
        { sourceId: "trigger", targetId: "first-action" },
        { sourceId: "first-action", targetId: "second-action" },
        { sourceId: "trigger", targetId: "independent-action" },
      ],
    });

    const secondAction = sections.find((section) => section.nodeId === "second-action");

    expect(secondAction?.upstreamSections.map((section) => section.nodeName)).toEqual(["Trigger", "First Action"]);
    expect(secondAction?.upstreamSections.map((section) => section.output)).toEqual([
      { trigger: true },
      { first: true },
    ]);
    expect(secondAction?.primaryInputNodeId).toBe("first-action");
  });

  it("keeps channel names when displaying multiple execution output channels", () => {
    const run: CanvasesCanvasRun = {
      rootEvent: {
        id: "event-1",
        nodeId: "trigger",
        createdAt: new Date().toISOString(),
        data: { trigger: true },
      },
    };

    const [, actionSection] = buildRunInspectorNodeSections({
      run,
      executions: [
        {
          id: "execution-1",
          nodeId: "action",
          outputs: {
            default: [{ status: "ok" }],
            report: [{ url: "https://example.com/report" }],
          },
          metadata: {},
        },
      ],
      workflowNodes: [
        { id: "trigger", name: "Trigger", type: "TYPE_TRIGGER", component: "github.onPullRequest" },
        { id: "action", name: "Action", type: "TYPE_ACTION", component: "test.action" },
      ],
    });

    expect(actionSection.outputSections).toEqual([
      {
        channel: "default",
        value: { status: "ok" },
        sizeKb: expect.any(String),
      },
      {
        channel: "report",
        value: { url: "https://example.com/report" },
        sizeKb: expect.any(String),
      },
    ]);
  });
});

function buildTriggerNode(component: string): ComponentsNode {
  return {
    id: "trigger-1",
    name: component,
    type: "TYPE_TRIGGER",
    component,
  };
}

function buildGithubCommitStatusRun(): CanvasesCanvasRun {
  return {
    rootEvent: {
      id: "event-1",
      nodeId: "trigger-1",
      channel: "default",
      createdAt: new Date().toISOString(),
      data: {
        type: "github.status",
        data: {
          state: "success",
          context: "ci/semaphoreci/pr: Operately - Build & Test",
          description: "The build passed on Semaphore 2.0.",
          sha: "43a3f0a1ac3ba8bc45e1b858c557094af374eddb",
          branches: [],
          repository: {
            full_name: "operately/operately",
          },
          sender: {
            login: "shiroyasha",
          },
          target_url: "https://operately.semaphoreci.com/workflows/5128f711",
          commit: {
            sha: "43a3f0a1ac3ba8bc45e1b858c557094af374eddb",
            html_url: "https://github.com/operately/operately/commit/43a3f0a1ac3ba8bc45e1b858c557094af374eddb",
            author: {
              login: "Rockyy174",
            },
            commit: {
              message: "Adjust skills",
            },
          },
        },
      },
    },
  };
}
