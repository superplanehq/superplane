import type {
  CanvasFoldersCanvasFolder,
  CanvasesCanvasSummary,
  ComponentsEdge,
  SuperplaneComponentsNode,
} from "@/api-client";

/** Shared with AppPage stories so home → app continuity is obvious. */
export const HOME_ORGANIZATION_ID = "3ee1aa47-3a60-4c1f-b645-0b9859ab91f8";
export const SOFTWARE_FACTORY_APP_ID = "9725f25b-2947-4022-82f9-acb20a616bf6";

const FOLDER_AUTOMATION_ID = "folder-automation";
const FOLDER_RELEASES_ID = "folder-releases";

function miniGraph(
  prefix: string,
  layout: Array<[number, number]>,
): { nodes: SuperplaneComponentsNode[]; edges: ComponentsEdge[] } {
  const nodes = layout.map(([x, y], index) => ({
    id: `${prefix}-n${index + 1}`,
    position: { x, y },
  })) as SuperplaneComponentsNode[];

  const edges = nodes.slice(0, -1).map((node, index) => ({
    sourceId: node.id,
    targetId: nodes[index + 1]!.id,
    channel: "default",
  })) as ComponentsEdge[];

  return { nodes, edges };
}

function makeCanvas(
  id: string,
  name: string,
  options: {
    folderId?: string;
    description?: string;
    starred?: boolean;
    starredAt?: string;
    createdAt?: string;
    createdByName?: string;
    graph?: { nodes: SuperplaneComponentsNode[]; edges: ComponentsEdge[] };
  } = {},
): CanvasesCanvasSummary {
  const graph =
    options.graph ??
    miniGraph(id.slice(0, 8), [
      [0, 0],
      [240, 40],
      [480, 0],
    ]);

  return {
    id,
    name,
    description: options.description,
    folderId: options.folderId,
    createdAt: options.createdAt ?? "2026-05-05T00:00:00Z",
    createdBy: { name: options.createdByName ?? "Storybook User" },
    starred: options.starred,
    starredAt: options.starredAt,
    nodes: graph.nodes,
    edges: graph.edges,
  } as CanvasesCanvasSummary;
}

function makeFolder(
  id: string,
  title: string,
  backgroundColor: string,
  canvasIds: string[],
): CanvasFoldersCanvasFolder {
  return {
    metadata: { id },
    spec: {
      title,
      backgroundColor,
      canvases: canvasIds.map((canvasId) => ({ id: canvasId })),
    },
  } as CanvasFoldersCanvasFolder;
}

const softwareFactoryGraph = miniGraph("sf", [
  [72, -216],
  [552, -216],
  [1104, 0],
  [1728, 0],
  [2304, 0],
  [3000, 0],
  [3624, -216],
  [4224, -336],
  [4848, -480],
]);

const canvases: CanvasesCanvasSummary[] = [
  makeCanvas(SOFTWARE_FACTORY_APP_ID, "Software Factory", {
    folderId: FOLDER_AUTOMATION_ID,
    description: "Issue → plan → PR → CI babysitting for factory-labeled work.",
    starred: true,
    starredAt: "2026-07-16T12:00:00Z",
    createdAt: "2026-06-01T10:00:00Z",
    graph: softwareFactoryGraph,
  }),
  makeCanvas("app-pr-risk-review", "PR Risk Review", {
    folderId: FOLDER_AUTOMATION_ID,
    description: "Scores pull requests and posts a risk summary.",
    createdAt: "2026-06-10T14:00:00Z",
    graph: miniGraph("prr", [
      [0, 0],
      [200, 80],
      [400, 0],
      [600, 120],
    ]),
  }),
  makeCanvas("app-docs-reviewer", "Docs Reviewer", {
    folderId: FOLDER_AUTOMATION_ID,
    description: "Reviews documentation changes on open PRs.",
    createdAt: "2026-06-12T09:30:00Z",
  }),
  makeCanvas("app-superplane-saas", "SuperPlane SaaS", {
    folderId: FOLDER_RELEASES_ID,
    description: "Production deployment pipeline console.",
    createdAt: "2026-05-20T08:00:00Z",
    graph: miniGraph("saas", [
      [0, 40],
      [180, 0],
      [360, 40],
      [540, 0],
      [720, 40],
    ]),
  }),
  makeCanvas("app-superplane-release", "SuperPlane Release", {
    folderId: FOLDER_RELEASES_ID,
    description: "Release status, in-flight cuts, and history.",
    createdAt: "2026-05-22T11:15:00Z",
  }),
  makeCanvas("app-clean-code", "Clean Code Assessment", {
    description: "Grades PRs and posts a clean-code report.",
    createdAt: "2026-06-24T22:37:20Z",
    graph: miniGraph("cca", [
      [0, 0],
      [220, -60],
      [440, 0],
      [660, 80],
    ]),
  }),
];

const folders: CanvasFoldersCanvasFolder[] = [
  makeFolder(FOLDER_AUTOMATION_ID, "Automation", "blue", [
    SOFTWARE_FACTORY_APP_ID,
    "app-pr-risk-review",
    "app-docs-reviewer",
  ]),
  makeFolder(FOLDER_RELEASES_ID, "Releases", "green", ["app-superplane-saas", "app-superplane-release"]),
];

export interface HomePageFixture {
  organizationId: string;
  organizationName: string;
  canvases: CanvasesCanvasSummary[];
  folders: CanvasFoldersCanvasFolder[];
}

export const defaultHomePageFixture: HomePageFixture = {
  organizationId: HOME_ORGANIZATION_ID,
  organizationName: "SuperPlane",
  canvases,
  folders,
};

/** Fresh org: no apps or folders — HomePage redirects to the create/onboarding screen. */
export const emptyHomePageFixture: HomePageFixture = {
  organizationId: HOME_ORGANIZATION_ID,
  organizationName: "Acme",
  canvases: [],
  folders: [],
};
