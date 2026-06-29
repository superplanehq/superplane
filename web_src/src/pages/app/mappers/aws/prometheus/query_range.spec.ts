import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "..";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./common";
import { queryRangeMapper } from "./query_range";

describe("queryRangeMapper.props", () => {
  it("shows workspace alias, query, and start metadata", () => {
    const props = queryRangeMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          workspace: "ws-abc123",
          query: "up",
          start: "2026-06-08T09:00:00Z",
          end: "2026-06-08T10:00:00Z",
          step: "1m",
        },
        metadata: { workspaceAlias: "metrics" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "activity", label: "metrics" }),
        expect.objectContaining({ icon: "search", label: "up" }),
        expect.objectContaining({ icon: "clock", label: "Start: 2026-06-08T09:00:00Z" }),
      ]),
    );
    expect(props.metadata).not.toEqual(expect.arrayContaining([expect.objectContaining({ label: "ws-abc123" })]));
  });
});

describe("queryRangeMapper.getExecutionDetails", () => {
  it("maps range query output", () => {
    const details = queryRangeMapper.getExecutionDetails(
      buildDetailsCtx({
        node: {
          configuration: {
            region: "us-east-1",
            workspace: "ws-abc123",
            query: "up",
            start: "2026-06-08T09:00:00Z",
            end: "2026-06-08T10:00:00Z",
            step: "1m",
          },
          metadata: { workspaceAlias: "metrics" },
        },
        execution: {
          outputs: {
            default: [
              buildOutput({
                resultType: "matrix",
                result: [{ metric: { job: "prometheus" }, values: [[1717846800, "1"]] }],
              }),
            ],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Executed At": new Date("2026-06-08T09:01:00Z").toLocaleString(),
      Alias: "metrics",
      "Result Type": "matrix",
      Results: "1",
    });
    expect(details["Workspace ID"]).toBeUndefined();
    expect(details.Query).toBeUndefined();
    expect(details.Start).toBeUndefined();
    expect(details.End).toBeUndefined();
    expect(details.Step).toBeUndefined();
  });
});

describe("eventStateRegistry.prometheus.queryRange", () => {
  it("keeps the success state label as success", () => {
    expect(eventStateRegistry["prometheus.queryRange"].getState(buildExecution())).toBe("success");
  });
});
