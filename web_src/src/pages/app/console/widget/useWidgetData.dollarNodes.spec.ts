import { describe, it, expect } from "vitest";

import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode } from "@/api-client";

import { DOLLAR_REWRITE_IDENTIFIER } from "./celExpr";
import { buildDollarNodes, buildNodeNameMap, collectRunRows, lastOutputData } from "./useWidgetData";

const NODES: SuperplaneComponentsNode[] = [
  { id: "node-deploy", name: "deploy-prod", type: "TYPE_ACTION" },
  { id: "node-build", name: "build", type: "TYPE_ACTION" },
  { id: "node-noname", type: "TYPE_ACTION" }, // no name on purpose
];

const DEPLOY_EXEC: CanvasesCanvasNodeExecution = {
  id: "exec-deploy",
  nodeId: "node-deploy",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  outputs: {
    default: [{ data: { url: "https://deploy.example.com", revision: "abc" } }],
  },
};

const BUILD_EXEC: CanvasesCanvasNodeExecution = {
  id: "exec-build",
  nodeId: "node-build",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  outputs: {
    artifacts: ["sha256:1", "sha256:2"], // raw (non-envelope) value
  },
};

const NAMELESS_EXEC: CanvasesCanvasNodeExecution = {
  id: "exec-nameless",
  nodeId: "node-noname",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
};

describe("lastOutputData", () => {
  it("returns undefined when outputs is missing or empty", () => {
    expect(lastOutputData(undefined)).toBeUndefined();
    expect(lastOutputData({})).toBeUndefined();
    expect(lastOutputData({ default: [] })).toBeUndefined();
  });

  it("prefers the `default` channel and unwraps the `data` envelope", () => {
    const data = lastOutputData({
      default: [{ data: { url: "https://final" } }, { data: { url: "https://later" } }],
      other: [{ data: { url: "https://other" } }],
    });
    expect(data).toEqual({ url: "https://later" });
  });

  it("falls back to the first channel when `default` is absent", () => {
    const data = lastOutputData({
      ci: [{ data: { passed: true } }],
    });
    expect(data).toEqual({ passed: true });
  });

  it("returns the raw event when it isn't envelope-shaped", () => {
    expect(lastOutputData({ artifacts: ["sha256:1", "sha256:2"] })).toBe("sha256:2");
    expect(lastOutputData({ scores: [42] })).toBe(42);
  });

  it("treats arrays as raw values (does not look for `.data` on arrays)", () => {
    const arr = [1, 2, 3];
    expect(lastOutputData({ default: [arr] })).toBe(arr);
  });
});

describe("buildDollarNodes", () => {
  const nameMap = buildNodeNameMap(NODES);

  it("returns an empty object for missing executions", () => {
    expect(buildDollarNodes(undefined, nameMap)).toEqual({});
    expect(buildDollarNodes([], nameMap)).toEqual({});
  });

  it("keys entries by node display name", () => {
    const out = buildDollarNodes([DEPLOY_EXEC, BUILD_EXEC], nameMap);
    expect(Object.keys(out).sort()).toEqual(["build", "deploy-prod"]);
  });

  it("falls back to the nodeId when the canvas node has no name", () => {
    const out = buildDollarNodes([NAMELESS_EXEC], nameMap);
    expect(out["node-noname"]).toBeDefined();
  });

  it("spreads the full execution and adds a `data` shortcut", () => {
    const out = buildDollarNodes([DEPLOY_EXEC], nameMap) as Record<string, Record<string, unknown>>;
    const entry = out["deploy-prod"];
    expect(entry.id).toBe("exec-deploy");
    expect(entry.outputs).toEqual({ default: [{ data: { url: "https://deploy.example.com", revision: "abc" } }] });
    expect(entry.data).toEqual({ url: "https://deploy.example.com", revision: "abc" });
  });

  it("ignores executions without a nodeId", () => {
    const exec = { id: "exec-orphan", state: "STATE_FINISHED" } as CanvasesCanvasNodeExecution;
    const out = buildDollarNodes([exec], nameMap);
    expect(out).toEqual({});
  });
});

describe("collectRunRows with dollar nodes", () => {
  const PAGES = [
    {
      runs: [
        {
          id: "run-1",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          rootEvent: { id: "event-1", nodeId: "node-deploy", data: { pr: 7 } },
        },
        {
          id: "run-2",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          rootEvent: { id: "event-2", nodeId: "node-deploy", data: {} },
        },
        {
          id: "run-3",
          state: "STATE_STARTED",
          result: "RESULT_UNKNOWN",
          // No rootEvent.id -- can't side-load executions for this row.
          rootEvent: { nodeId: "node-deploy", data: {} },
        },
      ],
    },
  ];

  it("attaches `$` and the rewrite alias to every row when no executions are loaded", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as Array<Record<string, unknown>>;
    for (const row of rows) {
      expect(row.$).toEqual({});
      expect(row[DOLLAR_REWRITE_IDENTIFIER]).toEqual({});
      // Both keys must point at the same map so literal-path and CEL paths
      // see identical data.
      expect(row.$).toBe(row[DOLLAR_REWRITE_IDENTIFIER]);
    }
  });

  it("populates `$` from the executions map keyed by rootEvent.id", () => {
    const executionsByRootEventId = new Map<string, CanvasesCanvasNodeExecution[]>([
      ["event-1", [DEPLOY_EXEC, BUILD_EXEC]],
    ]);
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10, executionsByRootEventId) as Array<
      Record<string, unknown>
    >;
    const dollarRun1 = rows[0].$ as Record<string, unknown>;
    expect(Object.keys(dollarRun1).sort()).toEqual(["build", "deploy-prod"]);
    // Run 2 has its own rootEvent.id but no entry in the map -- empty.
    expect(rows[1].$).toEqual({});
    // Run 3 has no rootEvent.id -- empty.
    expect(rows[2].$).toEqual({});
  });

  it("renders missing node refs as undefined (which the renderer formats as `-`)", () => {
    const executionsByRootEventId = new Map<string, CanvasesCanvasNodeExecution[]>([
      ["event-1", [DEPLOY_EXEC]], // build did not run for this run
    ]);
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10, executionsByRootEventId) as Array<
      Record<string, unknown>
    >;
    const dollar = rows[0].$ as Record<string, unknown>;
    expect(dollar["deploy-prod"]).toBeDefined();
    expect(dollar["build"]).toBeUndefined();
  });

  it("`$` and the rewrite alias on a single row reference the same map", () => {
    const executionsByRootEventId = new Map<string, CanvasesCanvasNodeExecution[]>([["event-1", [DEPLOY_EXEC]]]);
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10, executionsByRootEventId) as Array<
      Record<string, unknown>
    >;
    expect(rows[0].$).toBe(rows[0][DOLLAR_REWRITE_IDENTIFIER]);
  });
});
