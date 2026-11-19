import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import type { OrganizationsIntegration } from "@/api-client";
import { factoryIntakesKey, useFactoryIntakes } from "@/hooks/useFactoryIntakeData";
import { useIntegration } from "@/hooks/useIntegrations";
import { WebhookSetupListener } from "@/hooks/useWebhookSetupWebsocket";
import { IntegrationWebhookFailures } from "./IntegrationWebhookFailures";

let canUpdate = true;
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: (_resource: string, action: string) => action !== "update" || canUpdate }),
}));

const originalConfig = client.getConfig();
let queryClient: QueryClient;
let integration: OrganizationsIntegration;
let retryFails = false;
let requests: Request[];
const sockets: WebSocket[] = [];
const originalWebSocket = globalThis.WebSocket;

class WebhookSetupSocket extends originalWebSocket {
  constructor(url: string | URL, protocols?: string | string[]) {
    super(url, protocols);
    sockets.push(this);
  }
}

beforeEach(() => {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  canUpdate = true;
  retryFails = false;
  requests = [];
  sockets.length = 0;
  globalThis.WebSocket = WebhookSetupSocket;
  integration = {
    metadata: { id: "integration-1", name: "GitHub", integrationName: "github" },
    status: {
      state: "ready",
      webhookFailures: [
        {
          id: "hook-1",
          state: "failed",
          repositories: ["acme/api"],
          events: ["pull_request", "pull_request_review"],
          limitEvent: "pull_request",
          message: "GitHub allows 20 webhooks for the pull_request event in each repository.",
        },
      ],
      webhookSetups: [{ id: "hook-1", state: "failed", repositories: ["acme/api"], events: ["pull_request"] }],
    },
  };
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (input) => {
      const request = input instanceof Request ? input : new Request(String(input));
      requests.push(request);
      if (request.url.endsWith("/intakes")) {
        return Response.json({
          intakes: [{ id: "intake-1", webhookSetupState: integration.status!.webhookSetups![0].state }],
        });
      }
      if (request.url.includes("/resources")) {
        return Response.json({
          resources: [
            {
              id: "12",
              name: "#12 · old.example/webhook · pull_request",
              url: "https://github.com/acme/api/settings/hooks/12",
            },
          ],
        });
      }
      if (request.method === "POST") {
        if (retryFails) return Response.json({ message: "Unavailable" }, { status: 503 });
        integration.status!.webhookFailures![0].state = "pending";
        integration.status!.webhookSetups![0].state = "pending";
        return Response.json({});
      }
      return Response.json({ integration });
    },
  });
});

afterEach(() => {
  globalThis.WebSocket = originalWebSocket;
  queryClient.clear();
  client.setConfig(originalConfig);
});

function Details() {
  const { data } = useIntegration("org-1", "integration-1");
  useFactoryIntakes("org-1", "factory-1");
  return data ? <IntegrationWebhookFailures organizationId="org-1" integration={data} /> : null;
}

function renderDetails() {
  render(
    <QueryClientProvider client={queryClient}>
      <WebhookSetupListener organizationId="org-1" />
      <Details />
    </QueryClientProvider>,
  );
}

it("identifies hooks for the limited event and follows a retry through setup completion", async () => {
  renderDetails();
  await screen.findByRole("alert");
  expect(screen.getByRole("link", { name: "Open acme/api webhooks" })).toHaveAttribute(
    "href",
    "https://github.com/acme/api/settings/hooks",
  );
  fireEvent.click(screen.getByRole("button", { name: /Show existing webhooks/ }));
  const hook = await screen.findByRole("link", { name: /#12/ });
  expect(hook).toHaveAttribute("href", "https://github.com/acme/api/settings/hooks/12");
  const inventoryRequest = requests.find((request) => request.url.includes("/resources"));
  expect(inventoryRequest?.url).toContain("pull_request");

  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "Trying again" })).toBeDisabled());
  expect(
    requests.filter((request) => request.method === "POST").map((request) => new URL(request.url).pathname),
  ).toEqual(["/api/v1/organizations/org-1/integrations/integration-1/webhooks/hook-1/retry"]);
  const intakeKey = factoryIntakesKey("org-1", "factory-1");
  await waitFor(() =>
    expect(queryClient.getQueryData(intakeKey)).toEqual([{ id: "intake-1", webhookSetupState: "failed" }]),
  );
  integration.status!.webhookFailures = [];
  integration.status!.webhookSetups![0].state = "ready";
  const socket = sockets.at(-1);
  expect(typeof socket?.onmessage).toBe("function");
  act(() => {
    socket?.onmessage?.(new MessageEvent("message", { data: JSON.stringify({ event: "webhook_setup_changed" }) }));
  });
  await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
  await waitFor(() =>
    expect(queryClient.getQueryData(intakeKey)).toEqual([{ id: "intake-1", webhookSetupState: "ready" }]),
  );
  fireEvent.click(screen.getByText("Webhook setup"));
  expect(screen.getByText("Registered")).toBeVisible();
});

it("keeps the repair available if the retry request fails", async () => {
  retryFails = true;
  renderDetails();
  fireEvent.click(await screen.findByRole("button", { name: "Try again" }));
  await screen.findByText("SuperPlane could not retry webhook setup. Try again.");
  expect(screen.getByRole("button", { name: "Try again" })).toBeEnabled();
});

it("does not let a reader retry webhook setup", async () => {
  canUpdate = false;
  renderDetails();
  expect(await screen.findByRole("button", { name: "Try again" })).toBeDisabled();
  expect(requests.some((request) => request.method === "POST")).toBe(false);
});
