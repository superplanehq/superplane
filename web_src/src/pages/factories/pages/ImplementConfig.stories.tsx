import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import type { IntegrationsCapabilityDefinition, SuperplaneComponentsNode } from "@/api-client";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";
import { storybookSelectableLlmModels } from "@/pages/home/__fixtures__/hostedLlmModels";
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
const features: string[] = [];
const action = {
  name: "runnerSuperPlane",
  label: "Run SuperPlane Agent",
  configuration: PLANNING_REVIEW_RUNNER_FIELDS.filter((field) => field.name !== "credentials").map((field) =>
    field.name === "model"
      ? { ...field, type: "hosted-model", typeOptions: { hostedModel: { provider: HOSTED_MODEL_ALL_PROVIDERS } } }
      : field,
  ),
};
const models = [
  ...storybookSelectableLlmModels(),
  {
    source: { id: "hosted", name: "SuperPlane" },
    provider: { id: "openrouter", name: "OpenRouter" },
    model: { id: "x-ai/grok-4.7", name: "x-ai/grok-4.7" },
    key: "hosted::openrouter::x-ai/grok-4.7",
    label: "x-ai/grok-4.7",
  },
];
const pullRequestAction: IntegrationsCapabilityDefinition = {
  name: "github.createPullRequest",
  label: "Create Pull Request",
  type: "TYPE_ACTION",
  configuration: [
    { name: "draft", label: "Draft", type: "boolean", description: "Create the pull request as a draft." },
  ],
};
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
  http.get(`/api/v1/actions/${action.name}`, () => HttpResponse.json({ action })),
  http.get(`/api/v1/organizations/${organizationId}/selectable-llm-models`, () => HttpResponse.json({ models })),
  http.get(`/api/v1/organizations/${organizationId}/workspace-usage`, () =>
    HttpResponse.json({ defaultHostedProvider: "openrouter", defaultHostedModel: "x-ai/grok-4.7" }),
  ),
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
  http.get(`/api/v1/canvases/${canvasId}/runs`, () => HttpResponse.json({ runs: [], totalCount: 0 })),
  http.get(`/api/v1/factories/${factoryId}/orders`, () => HttpResponse.json({ orders: [], hasNextPage: false })),
];

function ImplementConfigPreview({
  page = true,
  draftPullRequests = false,
  executionTimeoutSeconds = 3600,
  additionalInstructions = "",
  failSave = false,
}: {
  page?: boolean;
  draftPullRequests?: boolean;
  executionTimeoutSeconds?: number;
  additionalInstructions?: string;
  failSave?: boolean;
}) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [shouldFailSave, setShouldFailSave] = useState(failSave);
  const { credentials: _credentials, ...agentConfiguration } = PLANNING_REVIEW_DRAFT.components[0].configuration;
  const [specNodes, setSpecNodes] = useState<SuperplaneComponentsNode[]>(() => [
    { id: "onrun-implement", name: "Start implementation", type: "TYPE_TRIGGER", component: "onRun" },
    {
      id: IMPLEMENT_AGENT_NODE_ID,
      name: "Implement From Task Description",
      type: "TYPE_ACTION",
      component: action.name,
      configuration: {
        ...structuredClone(agentConfiguration),
        model: "",
        executionTimeoutSeconds,
        additionalInstructions,
      },
      concurrency: { max: 5, key: "" },
    },
    {
      id: "create-pr",
      name: "Create Pull Request",
      type: "TYPE_ACTION",
      component: "github.createPullRequest",
      configuration: { draft: draftPullRequests },
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
          initialNodeId={IMPLEMENT_AGENT_NODE_ID}
          onClose={() => console.log("Close Implement settings")}
          onDelete={() => console.log("Delete Implement automation")}
          onSaveNode={(update) => {
            if (shouldFailSave) {
              setShouldFailSave(false);
              return Promise.reject(new Error("Could not save settings"));
            }
            const updates = Array.isArray(update) ? update : [update];
            console.log("Save Implement settings", updates);
            setSpecNodes((nodes) =>
              updates.reduce(
                (current, next) =>
                  current.map((node) =>
                    node.id === next.nodeId
                      ? { ...node, name: next.name, configuration: next.configuration, concurrency: next.concurrency }
                      : node,
                  ),
                nodes,
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
export const Configuration: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const picker = await canvas.findByTestId("field-model-hosted-model");
    await userEvent.click(picker);
    const page = within(canvasElement.ownerDocument.body);
    await expect(page.getByTestId("field-model-hosted-model-list")).toBeVisible();
    await expect(page.getByTestId("field-model-hosted-thinking")).toBeVisible();
    await userEvent.keyboard("{Escape}");
    const runsTab = canvas.getByTestId("merge-confidence-config-tab-runs");
    await userEvent.click(runsTab);
    await expect(runsTab).toHaveAttribute("aria-selected", "true");
    await expect(canvas.queryByRole("group", { name: "Default model" })).not.toBeInTheDocument();
    await userEvent.click(canvas.getByTestId("merge-confidence-config-tab-automation"));
    await expect(canvas.getByTestId("field-model-hosted-model")).toBeVisible();
  },
};
export const InvalidDuration: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const duration = await canvas.findByRole("spinbutton", { name: "Maximum run duration (minutes)" });
    await userEvent.clear(duration);
    await userEvent.tab();
    await expect(canvas.getByRole("alert")).toBeVisible();
    await expect(canvas.getByRole("alert")).toHaveTextContent("Enter a duration from 1 second to 1440 minutes.");
    await expect(canvas.getByTestId("implement-settings-save")).toBeDisabled();
  },
};

export const WithInstructions: Story = {
  args: {
    additionalInstructions:
      "Avoid new production dependencies unless the task requires them.\nFor API changes, include a request and response example in the pull request description.",
  },
};
export const InstructionsSaveError: Story = {
  args: { failSave: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const instructions = await canvas.findByRole("textbox", { name: "Additional instructions" });
    await userEvent.type(instructions, "Use existing test helpers.");
    await userEvent.click(canvas.getByTestId("implement-settings-save"));
    await expect(canvas.getByRole("alert")).toBeVisible();
  },
};
