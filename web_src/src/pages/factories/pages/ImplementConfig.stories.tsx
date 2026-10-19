import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { useState } from "react";
import type {
  FactoriesFactoryAgentResource,
  IntegrationsCapabilityDefinition,
  SuperplaneComponentsNode,
} from "@/api-client";
import {
  FEATURE_FACTORY_RUN_HISTORY,
  FEATURE_WORKSPACE_MCP,
  FEATURE_WORKSPACE_SKILLS,
} from "@/lib/experimentalFeatures";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { IMPLEMENT_AGENT_NODE_ID } from "../lib/columnAutomationHeadline";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import { PLANNING_REVIEW_RUNNER_FIELDS } from "./planningReviewRunnerFields";

const organizationId = "implement-config-org";
const factoryId = "implement-config-workspace";
const canvasId = "implement-config";
const features = [FEATURE_FACTORY_RUN_HISTORY, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS];
const action = { name: "runnerClaudeCode", label: "Run Claude Code", configuration: PLANNING_REVIEW_RUNNER_FIELDS };
const pullRequestAction: IntegrationsCapabilityDefinition = {
  name: "github.createPullRequest",
  label: "Create Pull Request",
  type: "TYPE_ACTION",
  configuration: [
    { name: "draft", label: "Draft", type: "boolean", description: "Create the pull request as a draft." },
  ],
};
const resources: FactoriesFactoryAgentResource[] = [
  {
    id: "sentry",
    name: "Sentry",
    kind: "KIND_MCP_SERVER",
    enabled: true,
    config: { url: "https://mcp.sentry.dev/mcp" },
  },
  {
    id: "clean-code",
    name: "clean-code",
    kind: "KIND_SKILL",
    enabled: true,
    description: "Use the repository coding standards.",
  },
];
const resourcePath = `/api/v1/factories/${factoryId}/agent-resources`;
function organizationResponse(enabledFeatures = features) {
  return HttpResponse.json({
    organization: {
      metadata: { id: organizationId, name: "SuperPlane" },
      spec: { enabledExperimentalFeatures: enabledFeatures },
    },
  });
}
const handlers = [
  http.get(`/api/v1/organizations/${organizationId}`, () => organizationResponse()),
  http.get("/account/experimental-features", () =>
    HttpResponse.json({ features: features.map((id) => ({ id, label: id, released: false })) }),
  ),
  http.get("/api/v1/actions", () => HttpResponse.json({ actions: [action] })),
  http.get("/api/v1/actions/runnerClaudeCode", () => HttpResponse.json({ action })),
  http.get("/api/v1/triggers", () => HttpResponse.json({ triggers: [{ name: "onRun", label: "On Run" }] })),
  http.get("/api/v1/widgets", () => HttpResponse.json({ widgets: [] })),
  http.get("/api/v1/integrations", () =>
    HttpResponse.json({ integrations: [{ name: "github", label: "GitHub", capabilities: [pullRequestAction] }] }),
  ),
  http.get(`/api/v1/organizations/${organizationId}/integrations`, () =>
    HttpResponse.json({
      integrations: [
        { metadata: { id: "github", name: "GitHub", integrationName: "github" }, status: { state: "ready" } },
      ],
    }),
  ),
  http.get(resourcePath, ({ request }) =>
    HttpResponse.json({
      resources: resources.filter((resource) => resource.kind === new URL(request.url).searchParams.get("kind")),
    }),
  ),
  http.get(`${resourcePath}/sentry/tools`, () =>
    HttpResponse.json({ tools: [{ name: "get_issue_details", description: "Read an issue and its stack trace." }] }),
  ),
  http.get(`/api/v1/canvases/${canvasId}/runs`, () => HttpResponse.json({ runs: [], totalCount: 0 })),
  http.get(`/api/v1/factories/${factoryId}/orders`, () => HttpResponse.json({ orders: [], hasNextPage: false })),
];

function ImplementConfigPreview({
  page = true,
  initialNodeId = IMPLEMENT_AGENT_NODE_ID,
}: {
  page?: boolean;
  initialNodeId?: string;
}) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [specNodes, setSpecNodes] = useState<SuperplaneComponentsNode[]>(() => [
    { id: "onrun-implement", name: "Start implementation", type: "TYPE_TRIGGER", component: "onRun" },
    {
      id: IMPLEMENT_AGENT_NODE_ID,
      name: "Implement From Task Description",
      type: "TYPE_ACTION",
      component: "runnerClaudeCode",
      configuration: structuredClone(PLANNING_REVIEW_DRAFT.components[0].configuration),
      concurrency: { max: 5, key: "" },
    },
    {
      id: "create-pr",
      name: "Create Pull Request",
      type: "TYPE_ACTION",
      component: "github.createPullRequest",
      configuration: { draft: true },
      integration: { id: "github" },
    },
  ]);
  const prepared = prepareData({
    workflow: {
      metadata: { id: canvasId },
      spec: {
        nodes: specNodes,
        edges: [
          { sourceId: "onrun-implement", targetId: IMPLEMENT_AGENT_NODE_ID, channel: "default" },
          { sourceId: IMPLEMENT_AGENT_NODE_ID, targetId: "create-pr", channel: "default" },
        ],
      },
    },
    triggers: [{ name: "onRun", label: "On Run" }],
    components: [action, pullRequestAction],
    nodeEventsMap: {},
    nodeExecutionsMap: {},
    nodeQueueItemsMap: {},
    workflowId: canvasId,
    queryClient: client,
    user: null,
    canvasMode: "live",
  });
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell initialPath={`/?automationView=${canvasId}`} className="h-screen bg-background">
        <MergeConfidenceConfigModal
          title="Implement"
          page={page}
          graph={{ nodes: prepared.nodes, edges: prepared.edges, specNodes, organizationId, factoryId }}
          canvasId={canvasId}
          factoryId={factoryId}
          factoryKey="SUPER"
          initialNodeId={initialNodeId}
          onClose={() => console.log("Close Implement settings")}
          onDelete={() => console.log("Delete Implement automation")}
          onSaveNode={(update) => {
            console.log("Save Implement settings", update);
            setSpecNodes((nodes) =>
              nodes.map((node) =>
                node.id === update.nodeId
                  ? { ...node, name: update.name, configuration: update.configuration, concurrency: update.concurrency }
                  : node,
              ),
            );
          }}
        />
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Pages/Implement configuration",
  component: ImplementConfigPreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen", msw: { handlers } },
} satisfies Meta<typeof ImplementConfigPreview>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Configuration: Story = {};
export const Popup: Story = { args: { page: false } };
export const PullRequestSettings: Story = { args: { initialNodeId: "create-pr" } };
export const FeatureDisabled: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get(`/api/v1/organizations/${organizationId}`, () =>
          organizationResponse(features.filter((id) => id !== FEATURE_FACTORY_RUN_HISTORY)),
        ),
        ...handlers,
      ],
    },
  },
};
export const EmptyResources: Story = {
  parameters: { msw: { handlers: [http.get(resourcePath, () => HttpResponse.json({ resources: [] })), ...handlers] } },
};
export const ResourceError: Story = {
  parameters: {
    msw: { handlers: [http.get(resourcePath, () => new HttpResponse(null, { status: 500 })), ...handlers] },
  },
};
