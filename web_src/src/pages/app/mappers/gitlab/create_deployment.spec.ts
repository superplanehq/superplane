import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { createDeploymentMapper } from "./create_deployment";

describe("createDeploymentMapper", () => {
  it("shows created deployment details", () => {
    const details = createDeploymentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          environment: "production",
          ref: "main",
          sha: "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
          status: "running",
        },
        outputs: {
          default: [
            {
              data: {
                id: 42,
                iid: 2,
                ref: "main",
                sha: "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
                status: "running",
                created_at: "2026-06-11T14:30:00Z",
                environment: { id: 9, name: "production", external_url: "https://prod.example.com" },
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Created At": expect.any(String),
      Status: "running",
      Environment: "production",
      Ref: "main",
      SHA: "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
      URL: "https://prod.example.com",
    });
  });

  it("handles missing outputs", () => {
    const details = createDeploymentMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", environment: "production", ref: "main" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Create Deployment",
    componentName: "gitlab.createDeployment",
    isCollapsed: false,
    metadata: {
      project: {
        id: 123,
        name: "felixgateru/hello-world",
        url: "https://gitlab.com/felixgateru/hello-world",
      },
    },
  };

  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      ...execution,
    },
  };
}
