import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { expect, vi } from "bun:test";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { dematerializeCanvasSpec, materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { TooltipProvider } from "@/ui/tooltip";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { reviewRequestCanvas } from "../__fixtures__/reviewRequestFixture";
import { ColumnAutomationViewHost } from "./ColumnAutomationViewPopup";

interface SettingsOptions {
  legacy?: boolean;
  failSave?: boolean;
  failReviewers?: boolean;
  failRetry?: boolean;
  initialPath?: string;
}
const canvasPath = "/api/v1/canvases/review-request-app";
const model = (id: string) => ({
  source: { id: "hosted", name: "SuperPlane" },
  provider: { id: "openai", name: "OpenAI" },
  model: { id, name: id },
  key: `hosted::openai::${id}`,
  label: `openai/${id}`,
});
const actions = [
  {
    name: "runnerSuperPlane",
    label: "Run SuperPlane Agent",
    configuration: [
      { name: "model", label: "Model", type: "hosted-model", typeOptions: { hostedModel: { provider: "all" } } },
    ],
  },
  { name: "github.ensurePullRequestReviewers", label: "Request reviewers", configuration: [] },
];
const decision = {
  status: "requested",
  rationale: "Ownership review is required for a cautious report.",
  reviewers: [{ login: "sam", reason: "Owns the changed package." }],
};

type Handler = (request: Request) => Response | Promise<Response>;
class SettingsServer {
  stored = reviewRequestCanvas();
  stagedYaml = "";
  reviewerRequests: string[] = [];
  retryRequests: string[] = [];
  run = {
    id: "run-1",
    versionId: "version-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: new Date(Date.now() - 60000).toISOString(),
    finishedAt: new Date().toISOString(),
    rootEvent: { id: "report-event", nodeId: "review-request-trigger", data: {} },
    executions: [
      { id: "selection-1", nodeId: "select-reviewers" },
      { id: "outcome-1", nodeId: "request-reviewers" },
    ],
  };
  handlers: Map<string, Handler>;

  constructor(private options: SettingsOptions) {
    if (options.legacy && this.stored.spec) {
      this.stored.spec.nodes = this.stored.spec.nodes?.filter(
        (node) => node.id !== "pr-check-pending" && !node.id?.startsWith("findings-"),
      );
    }
    this.handlers = new Map<string, Handler>([
      [`POST ${canvasPath}/triggers/review-request-trigger/events/report-event/reemit`, () => this.retry()],
      [`POST ${canvasPath}/staging/commit`, () => this.commit()],
      [`PUT ${canvasPath}/staging`, (request) => this.stage(request)],
      [`GET ${canvasPath}/staging`, () => this.stagingSummary()],
      [`GET ${canvasPath}/file`, (request) => this.file(request)],
      [`GET ${canvasPath}`, () => Response.json({ canvas: this.stored })],
      [
        `GET ${canvasPath}/versions/version-1`,
        () => Response.json({ version: { metadata: { id: "version-1" }, spec: this.stored.spec } }),
      ],
      [`GET ${canvasPath}/runs/run-1`, () => Response.json({ run: this.run })],
      [`GET ${canvasPath}/runs`, () => Response.json({ runs: [this.run] })],
      [`GET ${canvasPath}/events/report-event/executions`, () => Response.json({ executions: this.executions() })],
      ["GET /api/v1/organizations/org-1/integrations/github-1/resources", (request) => this.reviewers(request)],
    ]);
  }

  async respond(input: RequestInfo | URL, init?: RequestInit) {
    const request = new Request(input instanceof Request ? input : new URL(String(input), "http://localhost"), init);
    const path = new URL(request.url).pathname;
    const handler = this.handlers.get(`${request.method} ${path}`);
    return handler ? handler(request) : Response.json(catalogResponse(path));
  }

  private retry() {
    this.retryRequests.push("retry");
    if (this.options.failRetry) {
      this.options.failRetry = false;
      return Response.json({ message: "Could not retry selection. Try again." }, { status: 503 });
    }
    return Response.json({ eventId: "retry-event" });
  }

  private commit() {
    if (this.options.failSave) {
      this.options.failSave = false;
      return Response.json({ message: "Could not commit settings. Try again." }, { status: 500 });
    }
    this.stored = { ...this.stored, spec: dematerializeCanvasSpec(this.stagedYaml) ?? this.stored.spec };
    this.stagedYaml = "";
    return this.stagingSummary();
  }

  private async stage(request: Request) {
    const body = (await request.json()) as { operations: Array<{ content: string }> };
    this.stagedYaml = atob(body.operations[0].content);
    return this.stagingSummary();
  }

  private stagingSummary() {
    return Response.json({ stagingSummary: { hasStaging: Boolean(this.stagedYaml), stale: false } });
  }

  private file(request: Request) {
    const staged = new URL(request.url).searchParams.get("stage") === "true" && this.stagedYaml;
    return new Response(staged || materializeCanvasSpec(this.stored));
  }

  private reviewers(request: Request) {
    const repository = new URL(request.url).searchParams.get("repository") ?? "";
    this.reviewerRequests.push(repository);
    if (this.options.failReviewers) return Response.json({ message: "GitHub unavailable" }, { status: 503 });
    return Response.json({
      resources:
        repository === "acme/second"
          ? [
              { id: "alex", name: "alex" },
              { id: "sam", name: "sam" },
            ]
          : [{ id: "Alex", name: "Alex" }],
    });
  }

  private executions() {
    return [
      {
        id: "selection-1",
        runId: "run-1",
        nodeId: "select-reviewers",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        outputs: {
          passed: [{ data: { result: { rationale: "Agent decision before application." } } }],
        },
      },
      {
        id: "outcome-1",
        runId: "run-1",
        nodeId: "review-outcome",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        outputs: { default: [{ data: { reviewRequest: decision } }] },
      },
    ];
  }
}

function catalogResponse(path: string): unknown {
  const responses: Record<string, unknown> = {
    "/account/experimental-features": { features: [{ id: "factory_review_request", released: false }] },
    "/api/v1/organizations/org-1": {
      organization: { metadata: { id: "org-1" }, spec: { enabledExperimentalFeatures: [] } },
    },
    "/api/v1/factories/factory-1/agent-resources": {
      resources: [{ id: "repo-2", kind: "KIND_CONTEXT_REPO", repository: "acme/second", enabled: true }],
    },
    "/api/v1/actions": { actions },
    "/api/v1/actions/runnerSuperPlane": { action: actions[0] },
    "/api/v1/triggers": { triggers: [{ name: "onBroadcast", label: "On broadcast", configuration: [] }] },
    "/api/v1/widgets": { widgets: [] },
    "/api/v1/organizations/org-1/selectable-llm-models": { models: [model("gpt-5"), model("gpt-5.4")] },
    "/api/v1/organizations/org-1/workspace-usage": { defaultHostedProvider: "openai", defaultHostedModel: "gpt-5" },
  };
  return responses[path] ?? {};
}

export function renderSettings(options: SettingsOptions = {}) {
  const server = new SettingsServer(options);
  const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation((input, init) => server.respond(input, init));
  const onClose = vi.fn();
  const mount = () =>
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter initialEntries={[options.initialPath ?? "/"]}>
          <ThemeProvider>
            <TooltipProvider>
              <FactoriesLayoutContext.Provider
                value={{
                  organizationId: "org-1",
                  factoryId: "factory-1",
                  factoryKey: "RF",
                  factories: [],
                  factory: {
                    id: "factory-1",
                    onboarding: {
                      vcsIntegrationId: "github-1",
                      appRepository: "acme/main",
                      backlogRepository: "acme/main",
                    },
                  },
                  openCreateWorkOrder: vi.fn(),
                }}
              >
                <ColumnAutomationViewHost
                  organizationId="org-1"
                  factoryId="factory-1"
                  factoryKey="RF"
                  canvasId="review-request-app"
                  columnKey="verify"
                  title="Review request"
                  onClose={onClose}
                />
              </FactoriesLayoutContext.Provider>
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );
  let view = mount();
  return {
    get view() {
      return view;
    },
    reload: () => {
      view.unmount();
      view = mount();
      return view;
    },
    onClose,
    stored: () => server.stored,
    reviewerRequests: server.reviewerRequests,
    retryRequests: server.retryRequests,
    fetchMock,
  };
}

export async function waitForPopupDismissal() {
  await waitFor(() => expect(document.body.style.pointerEvents).not.toBe("none"));
}
