import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { deleteReleaseMapper } from "./delete_release";

describe("deleteReleaseMapper", () => {
  it("shows deleted release details, including a successful tag deletion", () => {
    const details = deleteReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          releaseStrategy: "specific",
          tagName: "v1.0.0",
          deleteTag: true,
        },
        outputs: {
          default: [
            {
              data: {
                tag_name: "v1.0.0",
                name: "Release 1.0.0",
                tag_deleted: true,
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      Tag: "v1.0.0",
      Name: "Release 1.0.0",
      "Tag Deleted": "Yes",
    });
  });

  it("surfaces a failed tag deletion instead of hiding it", () => {
    const details = deleteReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          releaseStrategy: "specific",
          tagName: "v1.0.0",
          deleteTag: true,
        },
        outputs: {
          default: [
            {
              data: {
                tag_name: "v1.0.0",
                name: "Release 1.0.0",
                tag_deleted: false,
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      Tag: "v1.0.0",
      Name: "Release 1.0.0",
      "Tag Deleted": "No",
    });
  });

  it("omits the Tag Deleted row when the output has no tag_deleted field", () => {
    const details = deleteReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: {
          project: "123",
          releaseStrategy: "specific",
          tagName: "v1.0.0",
          deleteTag: false,
        },
        outputs: {
          default: [
            {
              data: {
                tag_name: "v1.0.0",
                name: "Release 1.0.0",
              },
            },
          ],
        },
      }),
    );

    expect(details).toEqual({
      Tag: "v1.0.0",
      Name: "Release 1.0.0",
    });
  });

  it("handles missing outputs", () => {
    const details = deleteReleaseMapper.getExecutionDetails(
      buildDetailsContext({
        configuration: { project: "123", releaseStrategy: "specific", tagName: "v1.0.0" },
        outputs: {},
      }),
    );

    expect(details).toEqual({});
  });

  it("shows project, tag and delete-tag indicator in node metadata", () => {
    const context = buildDetailsContext({});
    const props = deleteReleaseMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.deleteRelease",
        label: "Delete Release",
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
      { icon: "trash-2", label: "Also deletes tag" },
    ]);
  });

  it("omits the delete-tag indicator when disabled", () => {
    const context = buildDetailsContext({});
    context.node.configuration = { project: "123", releaseStrategy: "specific", tagName: "v1.0.0", deleteTag: false };

    const props = deleteReleaseMapper.props({
      nodes: context.nodes,
      node: context.node,
      componentDefinition: {
        name: "gitlab.deleteRelease",
        label: "Delete Release",
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
});

function buildDetailsContext(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Delete Release",
    componentName: "gitlab.deleteRelease",
    isCollapsed: false,
    configuration: {
      project: "123",
      releaseStrategy: "specific",
      tagName: "v1.0.0",
      deleteTag: true,
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
