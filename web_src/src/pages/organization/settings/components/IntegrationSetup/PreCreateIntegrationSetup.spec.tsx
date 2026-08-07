import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type * as SdkGen from "@/api-client/sdk.gen";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { IntegrationSetup } from ".";

const { integrationsListIntegrations, organizationsDescribeIntegration, organizationsListIntegrations } = vi.hoisted(
  () => ({
    integrationsListIntegrations: vi.fn(),
    organizationsDescribeIntegration: vi.fn(),
    organizationsListIntegrations: vi.fn(),
  }),
);

vi.mock("@/api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal<typeof SdkGen>();
  return {
    ...actual,
    integrationsListIntegrations,
    organizationsDescribeIntegration,
    organizationsListIntegrations,
  };
});

type TestInitialEntry = string | { pathname: string; state?: unknown };

function renderIntegrationSetup(initialEntry: TestInitialEntry = "/org-123/settings/integrations/setup/github") {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route
            path="/:organizationId/settings/integrations/setup/:integrationName"
            element={<IntegrationSetup organizationId="org-123" />}
          />
          <Route
            path="/:organizationId/settings/integrations/:integrationId"
            element={<div data-testid="integration-details">Integration details</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PreCreateIntegrationSetup", () => {
  beforeEach(() => {
    integrationsListIntegrations.mockResolvedValue({
      data: {
        integrations: [{ name: "github", label: "GitHub", capabilities: [] }],
      },
    });
    organizationsListIntegrations.mockResolvedValue({ data: { integrations: [] } });
    organizationsDescribeIntegration.mockResolvedValue({ data: { integration: null } });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("lets users clear the default instance name before typing a replacement", async () => {
    const user = userEvent.setup();
    renderIntegrationSetup();

    const input = await screen.findByLabelText("Name");
    await waitFor(() => expect(input).toHaveValue("github"));

    await user.clear(input);
    await user.type(input, "production");

    expect(input).toHaveValue("production");
  });

  it("polls redirect prompt setup state and navigates when setup completes", async () => {
    organizationsDescribeIntegration
      .mockResolvedValueOnce({
        data: {
          integration: {
            metadata: {
              id: "integration-123",
              name: "github",
              integrationName: "github",
              updatedAt: "2026-05-06T20:32:18Z",
            },
            status: {
              state: "pending",
              setupState: {
                currentStep: {
                  name: "create-github-app",
                  type: "REDIRECT_PROMPT",
                  label: "Create GitHub App",
                  instructions: "Create the GitHub App in GitHub, then return to SuperPlane.",
                  redirectPrompt: {
                    url: "https://github.com/settings/apps/new",
                    method: "GET",
                  },
                },
                previousSteps: [],
              },
            },
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          integration: {
            metadata: {
              id: "integration-123",
              name: "github",
              integrationName: "github",
              updatedAt: "2026-05-06T20:33:18Z",
            },
            status: {
              state: "ready",
            },
          },
        },
      });

    renderIntegrationSetup({
      pathname: "/org-123/settings/integrations/setup/github",
      state: { integrationId: "integration-123" },
    });

    expect(await screen.findByRole("button", { name: /continue/i })).toBeInTheDocument();
    await waitFor(() => expect(organizationsDescribeIntegration).toHaveBeenCalledTimes(2), { timeout: 5000 });
    await waitFor(() => expect(screen.getByTestId("integration-details")).toBeInTheDocument());
  });
});
