import type {
  CanvasesCanvas,
  SuperplaneComponentsEdge as ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
  OrganizationsIntegration,
} from "@/api-client";
import type { BuildingBlock } from "@/ui/BuildingBlocksSidebar";

export function makeCanvas(overrides: Partial<CanvasesCanvas> = {}): CanvasesCanvas {
  return {
    metadata: { id: "canvas-1" },
    spec: {
      nodes: [],
      edges: [],
    },
    ...overrides,
  } as CanvasesCanvas;
}

export function makeEdge(overrides: Partial<ComponentsEdge> = {}): ComponentsEdge {
  return {
    sourceId: "source-id",
    targetId: "target-id",
    channel: "default",
    ...overrides,
  };
}

export function makeIntegration(overrides: Partial<OrganizationsIntegration> = {}): OrganizationsIntegration {
  return {
    metadata: { id: "integration-id", name: "Default Integration" },
    spec: { integrationName: "github" },
    status: { state: "ready" },
    ...overrides,
  } as OrganizationsIntegration;
}

export function makeComponentsNode(overrides: Partial<ComponentsNode> = {}): ComponentsNode {
  return {
    id: "node-1",
    name: "Node 1",
    type: "TYPE_ACTION",
    position: { x: 0, y: 0 },
    configuration: {},
    component: "noop",
    ...overrides,
  } as ComponentsNode;
}

export function makeBuildingBlock(overrides: Partial<BuildingBlock> = {}): BuildingBlock {
  return {
    name: "http.request",
    type: "component",
    ...overrides,
  };
}
