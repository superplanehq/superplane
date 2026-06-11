import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import {
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
