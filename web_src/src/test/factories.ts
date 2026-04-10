import type {
  BlueprintsBlueprint,
  CanvasesCanvas,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsNode,
  IntegrationsIntegrationDefinition,
  OrganizationsIntegration,
  TriggersTrigger,
  WidgetsWidget,
} from "@/api-client";
import { Registry } from "@/lib/index/registry";
import type { BuildingBlock } from "@/lib/index/types";

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
    type: "TYPE_COMPONENT",
    position: { x: 0, y: 0 },
    configuration: {},
    component: { name: "noop" },
    ...overrides,
  } as ComponentsNode;
}

export function makeBuildingBlock(overrides: Partial<BuildingBlock> = {}): BuildingBlock {
  return {
    name: "http.request",
    type: "TYPE_COMPONENT",
    ...overrides,
  } as BuildingBlock;
}

export function makeRegistry({
  triggers = [],
  components = [],
  blueprints = [],
  widgets = [],
  availableIntegrations = [],
}: {
  triggers?: TriggersTrigger[];
  components?: ComponentsComponent[];
  blueprints?: BlueprintsBlueprint[];
  widgets?: WidgetsWidget[];
  availableIntegrations?: IntegrationsIntegrationDefinition[];
} = {}): Registry {
  return new Registry({
    triggers,
    components,
    blueprints,
    widgets,
    availableIntegrations,
  });
}
