import { describe, expect, it } from "bun:test";
import { buildGitLabDetailsContext } from "./test_support";

import type { ExecutionDetailsContext, ExecutionInfo } from "../types";
import { updateReleaseMapper } from "./update_release";

describe("updateReleaseMapper", () => {
  it("shows updated release details", () => {
    const details = updateReleaseMapper.getExecutionDetails(
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
                description: "Updated notes",
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
    const details = updateReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", releaseStrategy: "specific", tagName: "v1.0.0" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project and tag in node metadata for specific tag strategy", () => {
    const context = buildDetailsContext({});
    const props = updateReleaseMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.updateRelease",
        label: "Update Release",
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

    const props = updateReleaseMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.updateRelease",
        label: "Update Release",
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
      name: "Update Release",
      componentName: "gitlab.updateRelease",
      configuration: {
        project: "123",
        releaseStrategy: "specific",
        tagName: "v1.0.0",
      },
    },
    execution,
  );
}
