import { describe, it, expect } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";

import { buildNodeNameMap, collectRunRows } from "./useWidgetData";

const NODES: SuperplaneComponentsNode[] = [
  { id: "trigger-a", name: "deploy-prod", type: "TYPE_TRIGGER" },
  { id: "trigger-b", name: "manual-run", type: "TYPE_TRIGGER" },
  { id: "trigger-c", type: "TYPE_TRIGGER" }, // no name on purpose
];

const PAGES = [
  {
    runs: [
      {
        id: "run-1",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        rootEvent: { nodeId: "trigger-a", data: { pr_number: 42 } },
      },
      {
        id: "run-2",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        rootEvent: { nodeId: "trigger-b", data: { branch: "main" } },
      },
      {
        id: "run-3",
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        rootEvent: { nodeId: "trigger-c", data: {} },
      },
      {
        id: "run-4",
        state: "STATE_FINISHED",
        result: "RESULT_CANCELLED",
        rootEvent: { nodeId: "trigger-unknown", data: { reason: "user cancelled" } },
      },
    ],
  },
];

type RunRow = {
  id: string;
  status: string;
  nodeName?: string;
  payload?: Record<string, unknown>;
  rootEvent?: { nodeId?: string; data?: Record<string, unknown> };
};

describe("collectRunRows status", () => {
  it("derives status from state/result", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows.map((r) => r.status)).toEqual(["running", "passed", "failed", "cancelled"]);
  });

  it("maps a finished state with no explicit result to passed", () => {
    const pages = [{ runs: [{ id: "run-x", state: "STATE_FINISHED", rootEvent: { nodeId: "trigger-a" } }] }];
    const rows = collectRunRows(pages, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[0]?.status).toBe("passed");
  });

  it("maps an unknown state with no result to unknown", () => {
    const pages = [{ runs: [{ id: "run-x", rootEvent: { nodeId: "trigger-a" } }] }];
    const rows = collectRunRows(pages, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[0]?.status).toBe("unknown");
  });
});

describe("collectRunRows nodeName resolution", () => {
  it("uses the canvas node name for the initiating node, not its id", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[0]?.nodeName).toBe("deploy-prod");
    expect(rows[1]?.nodeName).toBe("manual-run");
  });

  it("falls back to nodeId when the canvas node has no name", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[2]?.nodeName).toBe("trigger-c");
  });

  it("falls back to nodeId when the run's initiating node is no longer on the canvas", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[3]?.nodeName).toBe("trigger-unknown");
  });

  it("leaves nodeName undefined when the run has no rootEvent", () => {
    const pages = [{ runs: [{ id: "run-x", state: "STATE_STARTED" }] }];
    const rows = collectRunRows(pages, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[0]?.nodeName).toBeUndefined();
  });
});

describe("collectRunRows payload", () => {
  it("exposes rootEvent.data as the top-level payload alias", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[0]?.payload).toEqual({ pr_number: 42 });
    expect(rows[1]?.payload).toEqual({ branch: "main" });
  });

  it("preserves the raw rootEvent so nested dot paths still resolve", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as RunRow[];
    expect(rows[0]?.rootEvent?.data).toEqual({ pr_number: 42 });
    expect(rows[0]?.rootEvent?.nodeId).toBe("trigger-a");
  });
});

describe("collectRunRows limit", () => {
  it("stops iterating once `limit` rows have been collected", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 2) as RunRow[];
    expect(rows).toHaveLength(2);
    expect(rows.map((r) => r.id)).toEqual(["run-1", "run-2"]);
  });
});

describe("collectRunRows `$` map default", () => {
  it("defaults `$` to an empty object when no executions side-load is provided", () => {
    const rows = collectRunRows(PAGES, buildNodeNameMap(NODES), 10) as Array<Record<string, unknown>>;
    for (const row of rows) {
      expect(row.$).toEqual({});
    }
  });
});

describe("collectRunRows durationMs", () => {
  it("derives ms-elapsed from createdAt -> finishedAt", () => {
    const pages = [
      {
        runs: [
          {
            id: "run-1",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-01-01T12:00:00Z",
            finishedAt: "2026-01-01T12:05:00Z",
            rootEvent: { nodeId: "trigger-a", data: {} },
          },
        ],
      },
    ];
    const rows = collectRunRows(pages, buildNodeNameMap(NODES), 10) as Array<Record<string, unknown>>;
    expect(rows[0]?.durationMs).toBe(5 * 60 * 1000);
  });

  it("leaves durationMs undefined when finishedAt is missing", () => {
    const pages = [
      {
        runs: [
          {
            id: "run-x",
            state: "STATE_STARTED",
            createdAt: "2026-01-01T12:00:00Z",
            rootEvent: { nodeId: "trigger-a", data: {} },
          },
        ],
      },
    ];
    const rows = collectRunRows(pages, buildNodeNameMap(NODES), 10) as Array<Record<string, unknown>>;
    expect(rows[0]?.durationMs).toBeUndefined();
  });

  it("leaves durationMs undefined when createdAt is missing", () => {
    const pages = [
      {
        runs: [
          {
            id: "run-y",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            finishedAt: "2026-01-01T12:05:00Z",
            rootEvent: { nodeId: "trigger-a", data: {} },
          },
        ],
      },
    ];
    const rows = collectRunRows(pages, buildNodeNameMap(NODES), 10) as Array<Record<string, unknown>>;
    expect(rows[0]?.durationMs).toBeUndefined();
  });
});
