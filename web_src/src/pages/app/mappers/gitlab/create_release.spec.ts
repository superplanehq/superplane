import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { createReleaseMapper } from "./create_release";

describe("createReleaseMapper", () => {
  it("shows created release details", () => {
    const details = createReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          tagName: "v1.0.0",
          ref: "main",
        },
        outputs: {
          default: [
            {
              data: {
                tag_name: "v1.0.0",
                name: "Release 1.0.0",
                description: "First release",
                released_at: "2026-06-11T14:30:00Z",
                milestones: [{ title: "v1.0" }],
                _links: { self: "https://gitlab.com/felixgateru/hello-world/-/releases/v1.0.0" },
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      Tag: "v1.0.0",
      Name: "Release 1.0.0",
      "Released At": expect.any(String),
      Milestones: "v1.0",
      URL: "https://gitlab.com/felixgateru/hello-world/-/releases/v1.0.0",
    });
  });

  it("handles missing outputs", () => {
    const details = createReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", tagName: "v1.0.0" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project, tag and ref in node metadata", () => {
    const context = buildDetailsContext({});
    const props = createReleaseMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.createRelease",
        label: "Create Release",
        description: "",
        icon: "gitlab",
        color: "orange",
      },
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
    });

    expect(props.metadata).toEqual([
      { icon: "book", label: "felixgateru/hello-world" },
      { icon: "tag", label: "v1.0.0" },
      { icon: "git-branch", label: "main" },
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Create Release",
    componentName: "gitlab.createRelease",
    isCollapsed: false,
    configuration: {
      project: "123",
      tagName: "v1.0.0",
      ref: "main",
    },
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
