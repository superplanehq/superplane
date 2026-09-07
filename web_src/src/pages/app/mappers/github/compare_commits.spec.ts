import { describe, expect, it, vi } from "vitest";

import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { compareCommitsMapper } from "./compare_commits";

describe("compareCommitsMapper", () => {
  it("shows repository and comparison refs", () => {
    const props = compareCommitsMapper.props(buildBaseContext(buildNode()));

    expect(props.metadata).toEqual([
      { icon: "book", label: "testhq/hello" },
      { icon: "git-compare", label: "main → feature/3317" },
    ]);
  });

  it.each([
    [{ statuses: ["added", "modified"] }, "added, modified"],
    [{ paths: ["src/**"] }, "src/**"],
    [{ statuses: [] }, "All statuses"],
    [{ paths: [] }, "All paths"],
    [{ statuses: ["renamed"], paths: ["src/a-very-long-pattern-that-needs-truncation/**"] }, "2 filters"],
  ])("summarizes active filters", (filters, summary) => {
    const props = compareCommitsMapper.props(buildBaseContext(buildNode(filters)));

    expect(props.metadata).toContainEqual({ icon: "funnel", label: summary });
  });

  it("shows unfiltered execution details", () => {
    const details = compareCommitsMapper.getExecutionDetails(buildDetailsContext("matched", comparisonPayload(3, 3)));

    expect(details).toEqual({
      "Changed files": "3",
      "Base SHA": "base-sha",
      "Head SHA": "head-sha",
      "Comparison URL": "https://github.com/testhq/hello/compare/main...feature",
    });
  });

  it("shows filtered and empty execution details", () => {
    const details = compareCommitsMapper.getExecutionDetails(
      buildDetailsContext("unmatched", comparisonPayload(3, 0, { statuses: ["added"] })),
    );

    expect(details["Changed files"]).toBe("3");
    expect(details["Matched files"]).toBe("0");
    expect(details["Comparison URL"]).toBe("https://github.com/testhq/hello/compare/main...feature");
  });

  it("shows missing count data", () => {
    const details = compareCommitsMapper.getExecutionDetails(
      buildDetailsContext("matched", { comparison: {}, appliedFilters: { paths: ["src/**"] } }),
    );

    expect(details["Changed files"]).toBe("-");
    expect(details["Matched files"]).toBe("-");
  });

  it("ignores undeclared output channels", () => {
    const details = compareCommitsMapper.getExecutionDetails(buildDetailsContext("changed", comparisonPayload(1, 1)));

    expect(details).toEqual({});
  });
});

function buildNode(filters: Record<string, unknown> = {}): NodeInfo {
  return {
    id: "node-1",
    name: "Compare commits",
    componentName: "github.compareCommits",
    isCollapsed: false,
    metadata: { repository: { name: "testhq/hello" } },
    configuration: { repository: "testhq/hello", base: "main", head: "feature/3317", ...filters },
  };
}

function buildBaseContext(node: NodeInfo): ComponentBaseContext {
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "github.compareCommits",
      label: "Compare Commits",
      description: "",
      icon: "github",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: vi.fn() },
  };
}

function buildDetailsContext(channel: string, payload: unknown): ExecutionDetailsContext {
  const node = buildNode();
  const execution = {
    id: "execution-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    outputs: { [channel]: [{ data: payload }] },
  } as ExecutionInfo;
  return { nodes: [node], node, execution };
}

function comparisonPayload(
  changedFileCount: number,
  returnedFileCount: number,
  appliedFilters?: Record<string, unknown>,
) {
  return {
    comparison: {
      changedFileCount,
      baseSha: "base-sha",
      headSha: "head-sha",
      url: "https://github.com/testhq/hello/compare/main...feature",
    },
    appliedFilters,
    files: Array.from({ length: returnedFileCount }, (_, index) => ({ path: `file-${index}` })),
  };
}
