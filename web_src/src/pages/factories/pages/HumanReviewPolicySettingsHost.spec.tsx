import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";

import { HumanReviewPolicySettingsHost } from "./HumanReviewPolicySettingsHost";

const originalConfig = client.getConfig();
let queryClient: QueryClient;
let reviewersAvailable = false;

beforeEach(() => {
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  reviewersAvailable = false;
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (input) => {
      const url = new URL(input instanceof Request ? input.url : String(input));
      if (url.pathname === "/api/v1/factories/factory-1") {
        return Response.json({
          factory: {
            onboarding: { appRepository: "mytmlt/project", vcsIntegrationId: "github-1", vcsProvider: "github" },
          },
        });
      }
      if (
        url.pathname === "/api/v1/organizations/org-1/integrations/github-1/resources" &&
        url.searchParams.get("type") === "reviewer" &&
        url.searchParams.get("repository") === "mytmlt/project"
      ) {
        if (!reviewersAvailable) {
          return Response.json({ message: "GitHub is unavailable" }, { status: 503 });
        }
        return Response.json({ resources: [{ id: "mytmlt", name: "mytmlt", type: "reviewer" }] });
      }
      return new Response("", { status: 404 });
    },
  });
});

afterEach(() => {
  queryClient.clear();
  client.setConfig(originalConfig);
});

it("loads reviewers for the workspace repository and recovers a failed request", async () => {
  render(
    <QueryClientProvider client={queryClient}>
      <HumanReviewPolicySettingsHost
        node={{ id: "prepare-human-review", configuration: { enabled: false } }}
        organizationId="org-1"
        factoryId="factory-1"
        onSave={vi.fn()}
      />
    </QueryClientProvider>,
  );

  expect(screen.getByRole("combobox", { name: "Default reviewer" })).toBeDisabled();
  expect(await screen.findByRole("alert")).toBeInTheDocument();
  reviewersAvailable = true;
  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() => expect(screen.getByRole("combobox", { name: "Default reviewer" })).toBeEnabled());
  fireEvent.click(screen.getByRole("combobox", { name: "Default reviewer" }));
  expect(await screen.findByRole("option", { name: "@mytmlt" })).toBeInTheDocument();
});
