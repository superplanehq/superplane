import { describe, expect, it, vi } from "vitest";
import type {
  BlueprintsBlueprint,
  CanvasesCanvas,
  ComponentsComponent,
  IntegrationsIntegrationDefinition,
} from "@/api-client";
import type { LayoutEngine } from "@/lib/layout";
import { makeCanvas, makeComponentsNode, makeEdge, makeIntegration, makeRegistry } from "@/test/factories";
import { CanvasBuilder } from "./canvas-builder";

function makeComponent(overrides: Partial<ComponentsComponent> = {}): ComponentsComponent {
  return {
    name: "http.request",
    ...overrides,
  } as ComponentsComponent;
}

function makeAvailableIntegration(
  overrides: Partial<IntegrationsIntegrationDefinition> = {},
): IntegrationsIntegrationDefinition {
  return {
    name: "github",
    label: "GitHub",
    components: [],
    triggers: [],
    ...overrides,
  };
}

describe("CanvasBuilder", () => {
  it("adds component nodes with rounded position and resolves ready integration refs", async () => {
    const result = await new CanvasBuilder({
      canvas: makeCanvas(),
      registry: makeRegistry({
        availableIntegrations: [
          makeAvailableIntegration({
            name: "Git Hub",
            label: "Git Hub",
            components: [makeComponent({ name: "github.runWorkflow" })],
          }),
        ],
      }),
      integrations: [
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
    }).apply([
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

  it("creates annotation widgets with default annotation configuration", async () => {
    const result = await new CanvasBuilder({
      canvas: makeCanvas(),
      registry: makeRegistry({
        components: [makeComponent({ name: "annotation" })],
      }),
    }).apply([
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

  it("adds an edge for add_node source refs using explicit handle channels", async () => {
    const source = makeComponentsNode({
      id: "source-node",
      name: "Source",
      component: { name: "source.block" },
    });
    const result = await new CanvasBuilder({
      canvas: makeCanvas({
        spec: { nodes: [source], edges: [] },
      }),
      registry: makeRegistry({
        components: [makeComponent({ name: "source.block" }), makeComponent({ name: "target.block" })],
      }),
    }).apply([
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

  it("picks a success-oriented channel when source output has no default", async () => {
    const result = await new CanvasBuilder({
      canvas: makeCanvas(),
      registry: makeRegistry({
        components: [
          makeComponent({
            name: "source.component",
            outputChannels: [
              { name: "on_error", label: "On error" },
              { name: "on_success", label: "When successful" },
            ],
          }),
          makeComponent({ name: "target.component" }),
        ],
      }),
    }).apply([
      {
        type: "add_node",
        blockName: "source.component",
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

  it("deduplicates connect operations and disconnects only the explicit channel", async () => {
    const result = await new CanvasBuilder({
      canvas: makeCanvas({
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
      registry: makeRegistry({
        components: [
          makeComponent({
            name: "source.block",
            outputChannels: [
              { name: "default", label: "Default" },
              { name: "error", label: "On error" },
            ],
          }),
          makeComponent({ name: "target.block" }),
        ],
      }),
    }).apply([
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

  it("merges node config updates and allows renaming by node name refs", async () => {
    const result = await new CanvasBuilder({
      canvas: makeCanvas({
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
      registry: makeRegistry({
        components: [makeComponent({ name: "http.request" })],
      }),
    }).apply([
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

  it("deletes target nodes and all connected edges", async () => {
    const result = await new CanvasBuilder({
      canvas: makeCanvas({
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
      registry: makeRegistry({
        components: [makeComponent({ name: "http.request" })],
      }),
    }).apply([
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

  it("calls layout engine when passed to apply and new nodes are added", async () => {
    const layoutApply = vi.fn(
      async (
        workflow: CanvasesCanvas,
        _options?: {
          components?: ComponentsComponent[];
          blueprints?: BlueprintsBlueprint[];
        },
      ) => workflow,
    );
    const layoutEngine: LayoutEngine = {
      estimateNodeSize: vi.fn(() => ({ width: 420, height: 180 })),
      apply: layoutApply,
    };

    const result = await new CanvasBuilder({
      canvas: makeCanvas(),
      registry: makeRegistry({
        components: [makeComponent({ name: "github.runWorkflow" })],
      }),
    }).apply(
      [
        {
          type: "add_node",
          blockName: "github.runWorkflow",
          nodeName: "Deploy Workflow",
        },
      ],
      layoutEngine,
    );

    expect(layoutApply).toHaveBeenCalledTimes(1);
    const [layoutCanvas, options] = layoutApply.mock.calls[0];
    expect(layoutCanvas.spec?.nodes).toHaveLength(1);
    expect(options?.components?.map((component) => component.name)).toEqual(["github.runWorkflow"]);
    expect(options?.blueprints).toEqual([]);
    expect(result.spec?.nodes).toHaveLength(1);
  });

  it("does not call layout engine when no net new nodes were added", async () => {
    const layoutApply = vi.fn(async (workflow: CanvasesCanvas) => workflow);
    const layoutEngine: LayoutEngine = {
      estimateNodeSize: vi.fn(() => ({ width: 420, height: 180 })),
      apply: layoutApply,
    };

    await new CanvasBuilder({
      canvas: makeCanvas(),
      registry: makeRegistry({
        components: [makeComponent({ name: "github.runWorkflow" })],
      }),
    }).apply(
      [
        {
          type: "add_node",
          blockName: "github.runWorkflow",
          nodeName: "Deploy Workflow",
          nodeKey: "temporary",
        },
        {
          type: "delete_node",
          target: { nodeKey: "temporary" },
        },
      ],
      layoutEngine,
    );

    expect(layoutApply).not.toHaveBeenCalled();
  });
});
