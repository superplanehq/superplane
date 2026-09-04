import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes, MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ApiKeysApiKey } from "@/api-client/types.gen";
import { OrganizationSettingsPathsProvider } from "@/lib/organizationSettingsPaths";
import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsApiKeysPage } from "./FactorySettingsApiKeysPage";

const createMutateAsync = vi.fn();
const deleteMutateAsync = vi.fn();
const regenerateMutateAsync = vi.fn();
const updateMutateAsync = vi.fn();

let apiKeys: ApiKeysApiKey[] = [];
let canCreate = true;
let canDelete = true;
let canUpdate = true;

const SETTINGS_PATHS = {
  apiKeys: "/org-1/workspaces/RF/settings/organization/api-keys",
  apiKeyDetail: (id: string) => `/org-1/workspaces/RF/settings/organization/api-keys/${id}`,
  secrets: "/org-1/workspaces/RF/settings/organization/secrets",
  secretDetail: (id: string) => `/org-1/workspaces/RF/settings/organization/secrets/${id}`,
};

const CI_KEY: ApiKeysApiKey = {
  id: "key-1",
  name: "ci-deploy",
  description: "Deploy from CI",
  canvasIds: [],
  createdAt: "2026-03-12T10:00:00.000Z",
  createdByName: "Ada",
  hasToken: true,
};

const STAGING_KEY: ApiKeysApiKey = {
  id: "key-2",
  name: "staging-bot",
  description: "Run staging checks",
  canvasIds: ["canvas-1"],
  expiresAt: "2026-10-03T23:59:59.000Z",
  createdAt: "2026-08-04T14:30:00.000Z",
  createdByName: "Ada",
  hasToken: true,
};

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/hooks/useReportPageReady", () => ({
  useReportPageReady: vi.fn(),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) => {
      if (resource !== "api_keys") {
        return false;
      }
      if (action === "create") {
        return canCreate;
      }
      if (action === "delete") {
        return canDelete;
      }
      if (action === "update") {
        return canUpdate;
      }
      return true;
    },
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Demo" } } }),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvases: () => ({ data: [{ id: "canvas-1", name: "Refund app" }] }),
}));

vi.mock("@/hooks/useApiKeys", () => ({
  useAPIKeys: () => ({ data: apiKeys, isLoading: false }),
  useAPIKey: (_organizationId: string, id: string) => ({
    data: apiKeys.find((apiKey) => apiKey.id === id) ?? null,
    isLoading: false,
    error: null,
  }),
  useCreateAPIKey: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useDeleteAPIKey: () => ({ mutateAsync: deleteMutateAsync, isPending: false }),
  useRegenerateAPIKeyToken: () => ({ mutateAsync: regenerateMutateAsync, isPending: false }),
  useUpdateAPIKey: () => ({ mutateAsync: updateMutateAsync, isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPage(path = SETTINGS_PATHS.apiKeys) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[path]}>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: REFUND_FACTORY.id ?? "factory-1",
              factory: REFUND_FACTORY,
            }}
          >
            <OrganizationSettingsPathsProvider paths={SETTINGS_PATHS}>
              <Routes>
                <Route
                  path="/org-1/workspaces/RF/settings/organization/api-keys"
                  element={<FactorySettingsApiKeysPage />}
                />
                <Route
                  path="/org-1/workspaces/RF/settings/organization/api-keys/:id"
                  element={<FactorySettingsApiKeysPage />}
                />
              </Routes>
            </OrganizationSettingsPathsProvider>
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsApiKeysPage", () => {
  beforeEach(() => {
    apiKeys = [CI_KEY, STAGING_KEY];
    canCreate = true;
    canDelete = true;
    canUpdate = true;
    createMutateAsync.mockReset();
    deleteMutateAsync.mockReset();
    regenerateMutateAsync.mockReset();
    updateMutateAsync.mockReset();
    deleteMutateAsync.mockResolvedValue({});
    regenerateMutateAsync.mockResolvedValue({ data: { token: "sp_org_new" } });
    updateMutateAsync.mockResolvedValue({});
  });

  it("lists API keys in a searchable card like secrets", () => {
    renderPage();

    expect(screen.getByTestId("factory-settings-api-keys")).toBeInTheDocument();
    expect(screen.getByTestId("factory-settings-api-keys-search")).toHaveAttribute("placeholder", "Search API keys");
    expect(screen.getByTestId("factory-settings-api-keys-list")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /ci-deploy/ })).toBeInTheDocument();
    expect(screen.getByText("Organization-wide · Never expires")).toBeInTheDocument();
    expect(screen.queryByText("Deploy from CI")).not.toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Revoke" })).toHaveLength(2);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("filters API keys by name", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByTestId("factory-settings-api-keys-search"), "staging");

    expect(screen.getByRole("button", { name: /staging-bot/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /ci-deploy/ })).not.toBeInTheDocument();
  });

  it("shows an empty state when the organization has no API keys", () => {
    apiKeys = [];
    renderPage();

    expect(screen.getByTestId("factory-settings-api-keys-empty")).toHaveTextContent("No API keys yet.");
    expect(screen.getByTestId("api-key-create-empty")).toHaveTextContent("Create API key");
    expect(screen.queryByTestId("factory-settings-api-keys-search")).not.toBeInTheDocument();
  });

  it("opens the selected API key as a settings page", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: /ci-deploy/ }));

    expect(screen.getByTestId("factory-settings-api-key-detail")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "All API keys" })).toBeInTheDocument();
    expect(screen.getByLabelText("API key name")).toHaveValue("ci-deploy");
    expect(screen.getByText("Deploy from CI")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Regenerate token" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Revoke API key" })).toBeInTheDocument();
    expect(screen.queryByText("Organization API key details.")).not.toBeInTheDocument();
  });
});
