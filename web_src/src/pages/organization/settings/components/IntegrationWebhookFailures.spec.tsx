import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import type { OrganizationsIntegration } from "@/api-client";
import { integrationKeys, useIntegration } from "@/hooks/useIntegrations";
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

beforeEach(() => {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  canUpdate = true;
  retryFails = false;
  requests = [];
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
    },
  };
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (input) => {
      const request = input instanceof Request ? input : new Request(String(input));
      requests.push(request);
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
        return Response.json({});
      }
      return Response.json({ integration });
    },
  });
});

afterEach(() => {
  queryClient.clear();
  client.setConfig(originalConfig);
});

function Details() {
  const { data } = useIntegration("org-1", "integration-1");
  return data ? <IntegrationWebhookFailures organizationId="org-1" integration={data} /> : null;
}

function renderDetails() {
  render(
    <QueryClientProvider client={queryClient}>
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
  integration.status!.webhookFailures = [];
  await queryClient.invalidateQueries({ queryKey: integrationKeys.integration("org-1", "integration-1") });
  await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
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
