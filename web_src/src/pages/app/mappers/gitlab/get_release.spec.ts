import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
import { getReleaseMapper } from "./get_release";

describe("getReleaseMapper", () => {
  it("shows retrieved release details", () => {
    const details = getReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          releaseStrategy: "specific",
          tagName: "v1.0.0",
        },
        outputs: {
          default: [
            {
              data: {
                tag_name: "v1.0.0",
                name: "Release 1.0.0",
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
    const details = getReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", releaseStrategy: "specific", tagName: "v1.0.0" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project and tag in node metadata for specific tag strategy", () => {
    const context = buildDetailsContext({});
    const props = getReleaseMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.getRelease",
        label: "Get Release",
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
    ]);
  });

  it("shows 'Latest release' in node metadata for latest strategy", () => {
    const context = buildDetailsContext({});
    context.node.configuration = { project: "123", releaseStrategy: "latest" };

    const props = getReleaseMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.getRelease",
        label: "Get Release",
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
      { icon: "tag", label: "Latest release" },
    ]);
  });
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  return buildGitLabDetailsContext(
    {
      name: "Get Release",
      componentName: "gitlab.getRelease",
      configuration: {
        project: "123",
        releaseStrategy: "specific",
        tagName: "v1.0.0",
      },
    },
    execution,
  );
}
