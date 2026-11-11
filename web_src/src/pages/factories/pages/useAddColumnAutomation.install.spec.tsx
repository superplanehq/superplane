import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { type ReactNode } from "react";
import { afterEach, beforeAll, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import { materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { reviewRequestCanvas } from "../__fixtures__/reviewRequestFixture";
import { useAddColumnAutomation } from "./useAddColumnAutomation";

vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => true }) }));
beforeAll(() => client.setConfig({ baseUrl: "http://localhost" }));
afterEach(() => vi.restoreAllMocks());

it("installs Review request through the real installer and commits its materialized template", async () => {
  const requests: Array<{ method: string; path: string; body?: unknown }> = [];
  const materializePath = "/api/v1/factories/factory-1/app-templates/review-request:materialize";
  const responses: Record<string, unknown> = {
    "/account/experimental-features": { features: [{ id: "factory_review_request", released: false }] },
    "/api/v1/organizations/org-1": {
      organization: { metadata: { id: "org-1" }, spec: { enabledExperimentalFeatures: ["factory_review_request"] } },
    },
    "/api/v1/organizations/org-1/integrations/github-1": {
      integration: { metadata: { id: "github-1", name: "workspace-github" } },
    },
    "/api/v1/factories/factory-1/automations": {
      automations: [{ id: "merge-confidence-app", name: "Merge confidence", templateId: "risk-score" }],
    },
    "/api/v1/canvases": { canvas: { metadata: { id: "review-request-app", name: "Review request" } } },
    [materializePath]: {
      canvasYaml: materializeCanvasSpec(reviewRequestCanvas()),
      consoleYaml: "apiVersion: v1\nkind: Console\nspec: {}\n",
    },
  };
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const request = new Request(input instanceof Request ? input : new URL(String(input), "http://localhost"), init);
    const path = new URL(request.url).pathname;
    const body = request.method === "GET" ? undefined : await request.clone().json();
    requests.push({ method: request.method, path, body });
    return Response.json(responses[path] ?? {});
  });
  const { result } = renderHook(
    () =>
      useAddColumnAutomation({
        organizationId: "org-1",
        factoryId: "factory-1",
        factoryKey: "RF",
        lineId: "line-1",
        appRepository: "acme/main",
        backlogRepository: "acme/main",
        defaultBranch: "main",
        githubIntegrationId: "github-1",
        vcsProvider: "github",
        automationsFor: () => [],
      }),
    {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
          <MemoryRouter>{children}</MemoryRouter>
        </QueryClientProvider>
      ),
    },
  );
  act(() => result.current.openPicker("verify"));
  await waitFor(() => expect(result.current.catalog.some((entry) => entry.kind === "review-request")).toBe(true));
  const entry = result.current.catalog.find((entry) => entry.kind === "review-request")!;
  await act(async () => result.current.onSelect(entry));

  expect(
    requests.find((request) => request.method === "POST" && request.path === "/api/v1/canvases")?.body,
  ).toMatchObject({ name: "Review request", factoryId: "factory-1" });
  expect(requests.find((request) => request.path === materializePath)?.body).toMatchObject({
    appId: "review-request-app",
    integrations: [{ type: "github", id: "github-1", name: "workspace-github" }],
  });
  const writes = requests.filter((request) => ["PUT", "POST"].includes(request.method)).map((request) => request.path);
  expect(writes).toEqual([
    "/api/v1/canvases",
    materializePath,
    "/api/v1/canvases/review-request-app/staging",
    "/api/v1/canvases/review-request-app/staging/commit",
  ]);
  expect(result.current.pickerOpen).toBe(false);
});
