import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryWebhook } from "@/api-client/types.gen";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsWebhooksPage } from "./FactorySettingsWebhooksPage";
import { factorySettingsSectionRoutes } from "./factorySettingsSectionRoutes";

const createMutateAsync = vi.fn();
let webhooks: FactoriesFactoryWebhook[] = [];
let canRead = true;
let canUpdate = true;

const JIRA: FactoriesFactoryWebhook = {
  id: "webhook-1",
  name: "Jira",
  url: "https://superplane.test/api/v1/webhooks/webhook-1",
  authentication: "AUTHENTICATION_BEARER",
  automationCount: 1,
};

vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: vi.fn() }));
vi.mock("@/lib/toast", () => ({ showSuccessToast: vi.fn(), showErrorToast: vi.fn() }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (_resource: string, action: string) => (action === "read" ? canRead : canUpdate),
    hasPermission: (_resource: string, action: string) => (action === "read" ? canRead : canUpdate),
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useFactoryWebhooks", () => {
  const idle = () => ({ mutateAsync: vi.fn(), isPending: false });
  return {
    useFactoryWebhooks: () => ({ data: webhooks, isLoading: false, isError: false }),
    useFactoryWebhookMutations: () => ({
      create: { mutateAsync: createMutateAsync, isPending: false },
      update: idle(),
      remove: idle(),
      regenerateSecret: idle(),
    }),
  };
});

function renderPage(client?: QueryClient) {
  const page = (
    <TooltipProvider>
      <FactorySettingsLayoutContext.Provider
        value={{ organizationId: "org-1", factoryId: REFUND_FACTORY.id ?? "factory-1", factory: REFUND_FACTORY }}
      >
        {client ? (
          <Routes>
            <Route path="/:organizationId" element={<div>Organization home</div>} />
            <Route path="/:organizationId/workspaces/:factoryKey/settings">{factorySettingsSectionRoutes}</Route>
          </Routes>
        ) : (
          <FactorySettingsWebhooksPage />
        )}
      </FactorySettingsLayoutContext.Provider>
    </TooltipProvider>
  );
  return render(
    client ? (
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={["/org-1/workspaces/workspace-1/settings/workspace/webhooks"]}>
          {page}
        </MemoryRouter>
      </QueryClientProvider>
    ) : (
      page
    ),
  );
}

describe("FactorySettingsWebhooksPage", () => {
  beforeEach(() => {
    webhooks = [];
    canRead = true;
    canUpdate = true;
    createMutateAsync.mockReset();
  });

  for (const permission of ["none", "read", "update"]) {
    it(`enforces ${permission} permission on direct visits without optional features`, () => {
      canRead = permission !== "none";
      canUpdate = permission === "update";
      const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
      client.setQueryData(organizationKeys.details("org-1"), {
        spec: { enabledExperimentalFeatures: [] },
      });
      client.setQueryData(experimentalFeaturesKeys.registry(), {
        features: [{ id: "factory-as-code", released: false }],
      });
      const view = renderPage(client);
      if (canRead) {
        expect(screen.getByTestId("factory-webhooks")).toBeInTheDocument();
        if (canUpdate) {
          expect(screen.getByTestId("factory-webhook-create")).toBeEnabled();
        } else {
          expect(screen.getByTestId("factory-webhook-create")).toBeDisabled();
        }
      } else {
        expect(screen.getByTestId("permission-denied-page")).toBeInTheDocument();
        expect(screen.queryByTestId("factory-webhooks")).not.toBeInTheDocument();
      }
      view.unmount();
      client.clear();
    });
  }

  it("shows the secret once after the webhook is created", async () => {
    const user = userEvent.setup();
    createMutateAsync.mockResolvedValue({
      webhook: { ...JIRA, automationCount: 0, authentication: "AUTHENTICATION_HEADER_TOKEN", headerName: "X-Jira" },
      secret: "s3cr3t",
    });
    renderPage();

    expect(screen.getByTestId("factory-webhooks-empty")).toBeInTheDocument();
    await user.click(screen.getByTestId("factory-webhook-create"));
    await user.type(screen.getByTestId("factory-webhook-name"), "Jira");
    await user.click(screen.getByTestId("factory-webhook-submit"));

    expect(createMutateAsync).toHaveBeenCalledWith({
      name: "Jira",
      authentication: "AUTHENTICATION_BEARER",
      headerName: "",
    });
    const dialog = await screen.findByTestId("factory-webhook-secret-dialog");
    expect(screen.getByTestId("factory-webhook-secret")).toHaveTextContent("s3cr3t");
    expect(screen.getByTestId("factory-webhook-secret-url")).toHaveTextContent(JIRA.url ?? "");
    expect(dialog).toHaveTextContent('-H "X-Jira: $HEADER_TOKEN"');
  });

  it("blocks delete while an automation uses the webhook", () => {
    webhooks = [JIRA];
    renderPage();

    expect(screen.getByTestId("factory-webhook-url-webhook-1")).toHaveTextContent(JIRA.url ?? "");
    expect(screen.getByText(/Used by 1 automation/)).toBeInTheDocument();
    expect(screen.getByTestId("factory-webhook-delete-webhook-1")).toBeDisabled();
    expect(screen.getByTestId("factory-webhook-regenerate-webhook-1")).toBeEnabled();
  });
});
