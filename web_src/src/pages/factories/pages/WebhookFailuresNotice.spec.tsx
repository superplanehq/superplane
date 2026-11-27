import type { OrganizationsIntegration } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { integrationKeys } from "@/hooks/useIntegrations";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { WebhookFailuresNotice } from "./WebhookFailuresNotice";

let canRead = true;
let canUpdate = true;
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: (_resource: string, action: string) => (action === "read" ? canRead : canUpdate) }),
}));

const originalConfig = client.getConfig();
let queryClient: QueryClient;
let integrations: OrganizationsIntegration[];
let retryFailsFor: string | undefined;
let retriedWebhooks: string[];

beforeEach(() => {
  canRead = true;
  canUpdate = true;
  retryFailsFor = undefined;
  retriedWebhooks = [];
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  integrations = [
    {
      metadata: { id: "github-1", name: "GitHub", integrationName: "github" },
      status: {
        state: "ready",
        webhookFailures: [
          {
            id: "issues",
            state: "failed",
            repositories: ["acme/api"],
            events: ["issues"],
            limitEvent: "issues",
            message: "GitHub allows 20 webhooks for the issues event in each repository.",
          },
          {
            id: "reviews",
            state: "failed",
            repositories: ["acme/api"],
            events: ["pull_request", "pull_request_review"],
            limitEvent: "pull_request",
            message: "GitHub allows 20 webhooks for the pull_request event in each repository.",
          },
          {
            id: "reviews-duplicate",
            state: "failed",
            repositories: ["acme/api"],
            events: ["pull_request", "pull_request_review"],
            limitEvent: "pull_request",
            message: "GitHub allows 20 webhooks for the pull_request event in each repository.",
          },
          {
            id: "checks",
            state: "provisioning",
            repositories: ["acme/api"],
            events: ["check_run", "check_suite", "status"],
            message: "GitHub could not create the webhook. Check repository access and try again.",
          },
        ],
      },
    },
    {
      metadata: { id: "jira-1", name: "Jira", integrationName: "jira" },
      status: { state: "ready", webhookFailures: [{ id: "jira-issues", state: "failed" }] },
    },
  ];
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (input) => {
      const request = input instanceof Request ? input : new Request(input);
      if (request.method === "GET") return Response.json({ integrations });
      const webhookId = new URL(request.url).pathname.split("/").at(-2) ?? "";
      retriedWebhooks.push(webhookId);
      if (retryFailsFor === webhookId) return Response.json({ message: "Unavailable" }, { status: 503 });
      const failure = integrations
        .flatMap((integration) => integration.status?.webhookFailures ?? [])
        .find((webhook) => webhook.id === webhookId);
      if (!failure) return Response.json({ message: "Webhook not found" }, { status: 404 });
      failure.state = "pending";
      return Response.json({});
    },
  });
});

afterEach(() => {
  cleanup();
  queryClient.clear();
  client.setConfig({ ...originalConfig, fetch: originalConfig.fetch });
});

function notice() {
  return (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <WebhookFailuresNotice
          organizationId="org-1"
          integrationsBasePath="/org-1/workspaces/ENG/settings/organization/integrations"
        />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

it("groups GitHub failures without dropping retries and clears the notice when setup succeeds", async () => {
  render(notice());
  await screen.findByRole("alert");
  expect(screen.getAllByRole("heading", { name: "acme/api" })).toHaveLength(1);
  expect(screen.getByText("Events at the limit", { selector: "dt" })).toBeVisible();
  expect(screen.getByText("Issues, Pull requests", { selector: "dd" })).toBeVisible();
  expect(screen.getByText("Other affected events", { selector: "dt" })).toBeVisible();
  expect(
    screen.getByText("Pull request reviews, Check runs, Check suites, Commit statuses", { selector: "dd" }),
  ).toBeVisible();
  expect(screen.getAllByText(/GitHub allows 20 webhooks/)).toHaveLength(1);
  const accessError = integrations[0].status!.webhookFailures![3].message!;
  expect(screen.getAllByText(accessError)).toHaveLength(1);

  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "Trying again" })).toBeDisabled());
  await screen.findByText("Setup in progress for 4 of 4 webhooks.");
  expect(retriedWebhooks).toEqual(["issues", "reviews", "reviews-duplicate"]);
  expect(screen.getByText(accessError)).toBeVisible();

  integrations[0].status!.webhookFailures = [];
  await act(async () => {
    await queryClient.invalidateQueries({ queryKey: integrationKeys.connected("org-1") });
  });
  await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
});

it("identifies each affected connection and opens its settings without merging shared repositories", async () => {
  integrations[0].metadata!.name = "puppies-inc";
  integrations.push(
    {
      metadata: { id: "github-old", name: "puppies-inc-old", integrationName: "github" },
      status: {
        state: "ready",
        webhookFailures: [{ id: "old-hook", state: "failed", repositories: ["acme/api", "acme/web"] }],
      },
    },
    { metadata: { id: "github-healthy", name: "healthy", integrationName: "github" }, status: { state: "ready" } },
  );
  render(notice());

  for (const [name, id] of [
    ["puppies-inc", "github-1"],
    ["puppies-inc-old", "github-old"],
  ]) {
    const connection = within(await screen.findByRole("alert", { name: `${name} (GitHub)` }));
    expect(connection.getByRole("heading", { name: `${name} (GitHub)` })).toBeVisible();
    expect(connection.getByRole("link", { name: "Open integration settings" })).toHaveAttribute(
      "href",
      `/org-1/workspaces/ENG/settings/organization/integrations/${id}`,
    );
    expect(connection.getByRole("link", { name: "Open acme/api webhooks" })).toHaveAttribute(
      "href",
      "https://github.com/acme/api/settings/hooks",
    );
  }
  expect(screen.getAllByRole("link", { name: "Open integration settings" })).toHaveLength(2);
  expect(screen.queryByRole("alert", { name: "healthy (GitHub)" })).not.toBeInTheDocument();

  fireEvent.click(
    within(screen.getByRole("alert", { name: "puppies-inc-old (GitHub)" })).getByRole("button", { name: "Try again" }),
  );
  await waitFor(() => expect(retriedWebhooks).toEqual(["old-hook"]));
});

it("keeps the remaining failed webhook actionable after a partial retry failure", async () => {
  retryFailsFor = "reviews";
  render(notice());
  fireEvent.click(await screen.findByRole("button", { name: "Try again" }));
  await screen.findByRole("status");
  const retryButton = screen.getByRole("button", { name: "Try again" });
  expect(retryButton).toBeEnabled();
  expect(screen.getByRole("alert")).toBeVisible();
  expect(screen.getByText("Setup in progress for 3 of 4 webhooks.")).toBeVisible();

  retryFailsFor = undefined;
  fireEvent.click(retryButton);
  await waitFor(() => expect(screen.getByRole("button", { name: "Trying again" })).toBeDisabled());
  await screen.findByText("Setup in progress for 4 of 4 webhooks.");
  expect(retriedWebhooks).toEqual(["issues", "reviews", "reviews-duplicate", "reviews"]);
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
});

it("allows readers to inspect failures and hides cached failures when read access is removed", async () => {
  canUpdate = false;
  const { rerender } = render(notice());
  expect(await screen.findByRole("alert")).toBeVisible();
  expect(screen.getByRole("link", { name: "Open integration settings" })).toHaveAttribute(
    "href",
    "/org-1/workspaces/ENG/settings/organization/integrations/github-1",
  );
  expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();
  expect(retriedWebhooks).toHaveLength(0);

  canRead = false;
  rerender(notice());
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "Open integration settings" })).not.toBeInTheDocument();
});
