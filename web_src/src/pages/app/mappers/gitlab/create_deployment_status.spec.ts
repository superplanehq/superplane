import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { createDeploymentStatusMapper } from "./create_deployment_status";

describe("createDeploymentStatusMapper", () => {
  it("shows updated deployment details", () => {
    const details = createDeploymentStatusMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          deploymentId: "42",
          status: "success",
        },
        outputs: {
          default: [
            {
              data: {
                id: 42,
                iid: 2,
                ref: "main",
                sha: "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
                status: "success",
                created_at: "2026-06-11T14:30:00Z",
                updated_at: "2026-06-11T14:35:00Z",
                environment: { id: 9, name: "production", external_url: "https://prod.example.com" },
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      "Updated At": expect.any(String),
      Status: "success",
      Environment: "production",
      "Deployment ID": "42",
      Ref: "main",
      URL: "https://prod.example.com",
    });
  });

  it("falls back to created_at when updated_at is missing", () => {
    const details = createDeploymentStatusMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", deploymentId: "42", status: "success" },
        outputs: {
          default: [
            {
              data: {
                id: 42,
                status: "success",
                created_at: "2026-06-11T14:30:00Z",
              },
            },
          ],
        },
      }),
    );

    expect(details["Updated At"]).toEqual(new Date("2026-06-11T14:30:00Z").toLocaleString());
  });

  it("handles missing outputs", () => {
    const details = createDeploymentStatusMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", deploymentId: "42", status: "success" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Create Deployment Status",
    componentName: "gitlab.createDeploymentStatus",
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
