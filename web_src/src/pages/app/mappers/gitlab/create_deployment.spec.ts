import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
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
  return buildGitLabDetailsContext(
    {
      name: "Create Deployment",
      componentName: "gitlab.createDeployment",
    },
    execution,
  );
}
