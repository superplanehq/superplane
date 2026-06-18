import { describe, it, expect } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";

import { buildNodeNameMap, collectExecutionRows } from "./useWidgetData";

const NODES: SuperplaneComponentsNode[] = [
  { id: "node-a", name: "deploy-prod", type: "TYPE_TRIGGER" },
  { id: "node-b", name: "smoke-test", type: "TYPE_ACTION" },
  { id: "node-c", type: "TYPE_ACTION" }, // no name on purpose
];

const PAGES = [
  {
    runs: [
      {
        rootEvent: {
          data: { pr_number: 42, branch: "main" },
        },
        executions: [
          { id: "exec-1", nodeId: "node-a", state: "STATE_FINISHED", result: "RESULT_PASSED" },
          { id: "exec-2", nodeId: "node-b", state: "STATE_STARTED" },
          { id: "exec-3", nodeId: "node-c", state: "STATE_FINISHED", result: "RESULT_FAILED" },
          { id: "exec-4", nodeId: "node-unknown", state: "STATE_PENDING" },
        ],
      },
    ],
  },
];

type ExecutionRow = {
  id: string;
  nodeId: string;
  nodeName: string;
  status: string;
  payload?: Record<string, unknown>;
};

describe("buildNodeNameMap", () => {
  it("indexes nodes by id with name as the value", () => {
    const map = buildNodeNameMap(NODES);
    expect(map.get("node-a")).toBe("deploy-prod");
    expect(map.get("node-b")).toBe("smoke-test");
  });

  it("falls back to the node id when no name is present", () => {
    const map = buildNodeNameMap(NODES);
    expect(map.get("node-c")).toBe("node-c");
  });

  it("returns an empty map when the node list is missing", () => {
    expect(buildNodeNameMap(undefined).size).toBe(0);
  });
});

describe("collectExecutionRows nodeName resolution", () => {
  it("uses the canvas node name for each execution row, not the node id", () => {
    const rows = collectExecutionRows(PAGES, undefined, buildNodeNameMap(NODES), 10) as ExecutionRow[];
    expect(rows).toHaveLength(4);
    expect(rows[0]?.nodeId).toBe("node-a");
    expect(rows[0]?.nodeName).toBe("deploy-prod");
    expect(rows[1]?.nodeId).toBe("node-b");
    expect(rows[1]?.nodeName).toBe("smoke-test");
  });

  it("falls back to nodeId when the canvas node has no name", () => {
    const rows = collectExecutionRows(PAGES, undefined, buildNodeNameMap(NODES), 10) as ExecutionRow[];
    const namelessRow = rows.find((r) => r.nodeId === "node-c");
    expect(namelessRow?.nodeName).toBe("node-c");
  });

  it("falls back to nodeId when the execution's node is no longer on the canvas", () => {
    const rows = collectExecutionRows(PAGES, undefined, buildNodeNameMap(NODES), 10) as ExecutionRow[];
    const orphanRow = rows.find((r) => r.nodeId === "node-unknown");
    expect(orphanRow?.nodeName).toBe("node-unknown");
  });

  it("still resolves nodeName per row when a target node filter is applied", () => {
    const rows = collectExecutionRows(PAGES, "node-b", buildNodeNameMap(NODES), 10) as ExecutionRow[];
    expect(rows).toHaveLength(1);
    expect(rows[0]?.nodeName).toBe("smoke-test");
  });

  it("derives status from state/result", () => {
    const rows = collectExecutionRows(PAGES, undefined, buildNodeNameMap(NODES), 10) as ExecutionRow[];
    expect(rows.map((r) => r.status)).toEqual(["passed", "running", "failed", "pending"]);
  });

  it("attaches the run root event data as the row payload", () => {
    const rows = collectExecutionRows(PAGES, undefined, buildNodeNameMap(NODES), 10) as ExecutionRow[];
    for (const row of rows) {
      expect(row.payload).toEqual({ pr_number: 42, branch: "main" });
    }
  });

  it("leaves payload undefined when the run root event has no data", () => {
    const pages = [
      {
        runs: [
          {
            rootEvent: {},
            executions: [{ id: "exec-x", nodeId: "node-a", state: "STATE_STARTED" }],
          },
        ],
      },
    ];
    const rows = collectExecutionRows(pages, undefined, buildNodeNameMap(NODES), 10) as ExecutionRow[];
    expect(rows[0]?.payload).toBeUndefined();
  });
});
