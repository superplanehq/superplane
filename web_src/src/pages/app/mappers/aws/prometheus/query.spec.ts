import { describe, expect, it } from "vitest";

import { eventStateRegistry } from "..";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./common";
import { queryMapper } from "./query";

describe("queryMapper.props", () => {
  it("shows workspace alias, query, and region metadata", () => {
    const props = queryMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          workspace: "ws-abc123",
          query: "up",
        },
        metadata: { workspaceAlias: "metrics" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "activity", label: "metrics" }),
        expect.objectContaining({ icon: "search", label: "up" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
    expect(props.metadata).not.toEqual(expect.arrayContaining([expect.objectContaining({ label: "ws-abc123" })]));
  });
});

describe("queryMapper.getExecutionDetails", () => {
  it("maps instant query output", () => {
    const details = queryMapper.getExecutionDetails(
      buildDetailsCtx({
        node: {
          configuration: { region: "us-east-1", workspace: "ws-abc123", query: "up" },
          metadata: { workspaceAlias: "metrics" },
        },
        execution: {
          outputs: {
            default: [
              buildOutput({
                resultType: "vector",
                result: [{ metric: { job: "prometheus" }, value: [1717846800, "1"] }],
              }),
            ],
          },
        },
      }),
    );

    expect(details).toEqual({
      "Executed At": new Date("2026-06-08T09:01:00Z").toLocaleString(),
      Alias: "metrics",
      "Result Type": "vector",
      Results: "1",
    });
    expect(details["Workspace ID"]).toBeUndefined();
    expect(details.Query).toBeUndefined();
  });

  it("counts scalar query output as one result", () => {
    const details = queryMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [buildOutput({ resultType: "scalar", result: [1717846800, "1"] })],
          },
        },
      }),
    );

    expect(details.Results).toBe("1");
  });

  it("counts string query output as one result", () => {
    const details = queryMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          outputs: {
            default: [buildOutput({ resultType: "string", result: [1717846800, "ready"] })],
          },
        },
      }),
    );

    expect(details.Results).toBe("1");
  });
});

describe("eventStateRegistry.prometheus.query", () => {
  it("keeps the success state label as success", () => {
    expect(eventStateRegistry["prometheus.query"].getState(buildExecution())).toBe("success");
  });
});
