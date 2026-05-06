import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type * as SdkGen from "@/api-client/sdk.gen";
import { beforeEach, describe, expect, it, vi } from "vitest";
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

function renderIntegrationSetup() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/org-123/settings/integrations/setup/github"]}>
        <Routes>
          <Route
            path="/:organizationId/settings/integrations/setup/:integrationName"
            element={<IntegrationSetup organizationId="org-123" />}
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

  it("lets users clear the default instance name before typing a replacement", async () => {
    const user = userEvent.setup();
    renderIntegrationSetup();

    const input = await screen.findByLabelText("Name");
    await waitFor(() => expect(input).toHaveValue("github"));

    await user.clear(input);
    await user.type(input, "production");

    expect(input).toHaveValue("production");
  });
});
