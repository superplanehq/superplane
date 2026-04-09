import { describe, expect, it } from "vitest";
import type { BuildingBlock, BuildingBlockCategory } from "@/ui/BuildingBlocksSidebar";
import { makeBuildingBlock, makeCanvas, makeComponentsNode, makeEdge, makeIntegration } from "@/test/factories";
import { CanvasBuilder } from "./canvas-builder";

function makeCategories(...blocks: BuildingBlock[]): BuildingBlockCategory[] {
  return [{ name: "all", blocks }];
}

describe("CanvasBuilder", () => {
  it("adds component nodes with rounded position and resolves ready integration refs", () => {
    const result = new CanvasBuilder(
      makeCanvas(),
      makeCategories(
        makeBuildingBlock({
          name: "github.runWorkflow",
          integrationName: "Git Hub",
        }),
      ),
      [
        makeIntegration({
          metadata: { id: "pending-id", name: "Pending Integration" },
          spec: { integrationName: "git_hub" },
          status: { state: "provisioning" },
        }),
        makeIntegration({
          metadata: { id: "ready-id", name: "Ready Integration" },
          spec: { integrationName: "github" },
          status: { state: "ready" },
        }),
      ],
    ).apply([
      {
        type: "add_node",
        blockName: "github.runWorkflow",
        nodeName: "Deploy Workflow",
        position: { x: 10.6, y: -2.8 },
        configuration: { repo: "superplane" },
      },
    ]);

    expect(result.spec?.nodes).toHaveLength(1);
    const node = result.spec?.nodes?.[0];
    expect(node).toMatchObject({
      name: "Deploy Workflow",
      type: "TYPE_COMPONENT",
      component: { name: "github.runWorkflow" },
      configuration: { repo: "superplane" },
      position: { x: 11, y: -3 },
      integration: { id: "ready-id", name: "Ready Integration" },
    });
  });

  it("creates annotation widgets with default annotation configuration", () => {
    const result = new CanvasBuilder(makeCanvas(), makeCategories(makeBuildingBlock({ name: "annotation" }))).apply([
      {
        type: "add_node",
        blockName: "annotation",
        nodeName: "Important note",
        configuration: { text: "should be ignored", color: "red" },
      },
    ]);

    expect(result.spec?.nodes).toHaveLength(1);
    expect(result.spec?.nodes?.[0]).toMatchObject({
      name: "Important note",
      type: "TYPE_WIDGET",
      widget: { name: "annotation" },
      configuration: { text: "", color: "yellow" },
    });
  });

  it("adds an edge for add_node source refs using explicit handle channels", () => {
    const source = makeComponentsNode({
      id: "source-node",
      name: "Source",
      component: { name: "source.block" },
    });
    const result = new CanvasBuilder(
      makeCanvas({
        spec: { nodes: [source], edges: [] },
      }),
      makeCategories(makeBuildingBlock({ name: "source.block" }), makeBuildingBlock({ name: "target.block" })),
    ).apply([
      {
        type: "add_node",
        blockName: "target.block",
        nodeName: "Target",
        source: {
          nodeId: "source-node",
          handleId: "error",
        },
      },
    ]);

    expect(result.spec?.edges).toHaveLength(1);
    const targetNode = result.spec?.nodes?.find((node) => node.name === "Target");
    expect(result.spec?.edges?.[0]).toEqual({
      sourceId: "source-node",
      targetId: targetNode?.id,
      channel: "error",
    });
  });

  it("picks a success-oriented channel when source output has no default", () => {
    const result = new CanvasBuilder(
      makeCanvas(),
      makeCategories(
        makeBuildingBlock({
          name: "source.trigger",
          type: "trigger",
          outputChannels: [
            { name: "on_error", label: "On error" },
            { name: "on_success", label: "When successful" },
          ],
        }),
        makeBuildingBlock({ name: "target.component" }),
      ),
    ).apply([
      {
        type: "add_node",
        blockName: "source.trigger",
        nodeName: "Source Trigger",
        nodeKey: "source",
      },
      {
        type: "add_node",
        blockName: "target.component",
        nodeName: "Target Action",
        source: {
          nodeKey: "source",
        },
      },
    ]);

    expect(result.spec?.edges).toHaveLength(1);
    expect(result.spec?.edges?.[0]?.channel).toBe("on_success");
  });

  it("deduplicates connect operations and disconnects only the explicit channel", () => {
    const result = new CanvasBuilder(
      makeCanvas({
        spec: {
          nodes: [
            makeComponentsNode({
              id: "source",
              name: "Source",
              component: { name: "source.block" },
            }),
            makeComponentsNode({
              id: "target",
              name: "Target",
              component: { name: "target.block" },
            }),
          ],
          edges: [],
        },
      }),
      makeCategories(
        makeBuildingBlock({
          name: "source.block",
          outputChannels: [
            { name: "default", label: "Default" },
            { name: "error", label: "On error" },
          ],
        }),
        makeBuildingBlock({ name: "target.block" }),
      ),
    ).apply([
      {
        type: "connect_nodes",
        source: { nodeId: "source" },
        target: { nodeId: "target" },
      },
      {
        type: "connect_nodes",
        source: { nodeId: "source" },
        target: { nodeId: "target" },
      },
      {
        type: "connect_nodes",
        source: { nodeId: "source", handleId: "error" },
        target: { nodeId: "target" },
      },
      {
        type: "disconnect_nodes",
        source: { nodeId: "source", handleId: "error" },
        target: { nodeId: "target" },
      },
    ]);

    expect(result.spec?.edges).toEqual([
      {
        sourceId: "source",
        targetId: "target",
        channel: "default",
      },
    ]);
  });

  it("merges node config updates and allows renaming by node name refs", () => {
    const result = new CanvasBuilder(
      makeCanvas({
        spec: {
          nodes: [
            makeComponentsNode({
              id: "worker-id",
              name: "Worker",
              configuration: { existing: "value", retries: 2 },
            }),
          ],
          edges: [],
        },
      }),
      makeCategories(makeBuildingBlock({ name: "http.request" })),
    ).apply([
      {
        type: "update_node_config",
        target: { nodeName: "Worker" },
        nodeName: "Worker Updated",
        configuration: { retries: 3, timeout: 60 },
      },
    ]);

    expect(result.spec?.nodes?.[0]).toMatchObject({
      id: "worker-id",
      name: "Worker Updated",
      configuration: { existing: "value", retries: 3, timeout: 60 },
    });
  });

  it("deletes target nodes and all connected edges", () => {
    const result = new CanvasBuilder(
      makeCanvas({
        spec: {
          nodes: [
            makeComponentsNode({ id: "a", name: "A" }),
            makeComponentsNode({ id: "b", name: "B" }),
            makeComponentsNode({ id: "c", name: "C" }),
          ],
          edges: [
            makeEdge({ sourceId: "a", targetId: "b" }),
            makeEdge({ sourceId: "b", targetId: "c" }),
            makeEdge({ sourceId: "a", targetId: "c" }),
          ],
        },
      }),
      makeCategories(makeBuildingBlock({ name: "http.request" })),
    ).apply([
      {
        type: "delete_node",
        target: { nodeId: "b" },
      },
    ]);

    expect(result.spec?.nodes?.map((node) => node.id)).toEqual(["a", "c"]);
    expect(result.spec?.edges).toEqual([
      {
        sourceId: "a",
        targetId: "c",
        channel: "default",
      },
    ]);
  });
});
