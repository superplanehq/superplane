import { describe, expect, it } from "vitest";

import { graphqlMapper } from "./graphql";
import type { ExecutionDetailsContext, NodeInfo, OutputPayload } from "./types";

function buildNode(): NodeInfo {
  return {
    id: "node-1",
    name: "GraphQL",
    componentName: "graphql",
    isCollapsed: false,
    configuration: {},
    metadata: {},
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "json",
    timestamp: new Date().toISOString(),
    data,
  };
}

describe("graphqlMapper.getExecutionDetails", () => {
  it("includes GraphQL errors from the response body", () => {
    const node = buildNode();
    const ctx: ExecutionDetailsContext = {
      nodes: [node],
      node,
      execution: {
        id: "exec-1",
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        resultReason: "RESULT_REASON_OK",
        resultMessage: "",
        metadata: {},
        configuration: {},
        rootEvent: undefined,
        outputs: {
          failure: [
            buildOutput({
              status: 200,
              body: {
                errors: [
                  {
                    message: "Expected one of SCHEMA, SCALAR.",
                  },
                ],
              },
            }),
          ],
        },
      },
    };

    expect(graphqlMapper.getExecutionDetails(ctx)).toEqual({
      Response: "200",
      Errors: "Expected one of SCHEMA, SCALAR.",
    });
  });
});
