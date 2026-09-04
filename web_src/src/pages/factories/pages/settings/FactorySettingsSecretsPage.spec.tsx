import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SuperplaneSecretsSecret } from "@/api-client";
import { OrganizationSettingsPathsProvider } from "@/lib/organizationSettingsPaths";
import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsSecretsPage } from "./FactorySettingsSecretsPage";

const setKeyMutateAsync = vi.fn();
const deleteKeyMutateAsync = vi.fn();
const deleteSecretMutateAsync = vi.fn();
const updateNameMutateAsync = vi.fn();

let secrets: SuperplaneSecretsSecret[] = [];

const SETTINGS_PATHS = {
  apiKeys: "/org-1/workspaces/RF/settings/organization/api-keys",
  apiKeyDetail: (id: string) => `/org-1/workspaces/RF/settings/organization/api-keys/${id}`,
  secrets: "/org-1/workspaces/RF/settings/organization/secrets",
  secretDetail: (id: string) => `/org-1/workspaces/RF/settings/organization/secrets/${id}`,
};

const SECRET_WITH_FOO: SuperplaneSecretsSecret = {
  metadata: {
    id: "secret-1",
    name: "deploy-secret",
    createdAt: "2026-08-04T14:30:00.000Z",
  },
  spec: {
    provider: "PROVIDER_LOCAL",
    local: {
      data: {
        FOO: "value-1",
      },
    },
  },
};

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/hooks/useReportPageReady", () => ({
  useReportPageReady: vi.fn(),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: () => true,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Demo" } } }),
}));

vi.mock("@/hooks/useSecrets", () => ({
  useSecrets: () => ({ data: secrets, isLoading: false }),
  useSecret: (_domainId: string, _domainType: string, secretId: string) => ({
    data: secrets.find((secret) => secret.metadata?.id === secretId) ?? null,
    isLoading: false,
    error: null,
  }),
  useSetSecretKey: () => ({ mutateAsync: setKeyMutateAsync, isPending: false }),
  useDeleteSecretKey: () => ({ mutateAsync: deleteKeyMutateAsync, isPending: false }),
  useDeleteSecret: () => ({ mutateAsync: deleteSecretMutateAsync, isPending: false }),
  useUpdateSecretName: () => ({ mutateAsync: updateNameMutateAsync, isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

import { showErrorToast, showSuccessToast } from "@/lib/toast";

function renderPage(path: string) {
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
                  path="/org-1/workspaces/RF/settings/organization/secrets"
                  element={<FactorySettingsSecretsPage />}
                />
                <Route
                  path="/org-1/workspaces/RF/settings/organization/secrets/:secretId"
                  element={<FactorySettingsSecretsPage />}
                />
              </Routes>
            </OrganizationSettingsPathsProvider>
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsSecretsPage - add key", () => {
  beforeEach(() => {
    secrets = [SECRET_WITH_FOO];
    setKeyMutateAsync.mockReset();
    deleteKeyMutateAsync.mockReset();
    deleteSecretMutateAsync.mockReset();
    updateNameMutateAsync.mockReset();
    setKeyMutateAsync.mockResolvedValue({});
    vi.mocked(showErrorToast).mockReset();
    vi.mocked(showSuccessToast).mockReset();
  });

  it("shows an error toast and does not call the mutation when the key name already exists", async () => {
    const user = userEvent.setup();
    renderPage("/org-1/workspaces/RF/settings/organization/secrets/secret-1");

    await user.type(screen.getByLabelText("Key name"), "FOO");
    await user.type(screen.getByLabelText("Value"), "new-value");
    await user.click(screen.getByRole("button", { name: "Add key" }));

    expect(showErrorToast).toHaveBeenCalledWith("Key already exists");
    expect(setKeyMutateAsync).not.toHaveBeenCalled();
    expect(showSuccessToast).not.toHaveBeenCalled();
  });

  it("adds a new key and shows a success toast when the key name is unique", async () => {
    const user = userEvent.setup();
    renderPage("/org-1/workspaces/RF/settings/organization/secrets/secret-1");

    await user.type(screen.getByLabelText("Key name"), "BAR");
    await user.type(screen.getByLabelText("Value"), "new-value");
    await user.click(screen.getByRole("button", { name: "Add key" }));

    expect(setKeyMutateAsync).toHaveBeenCalledWith({ keyName: "BAR", value: "new-value" });
    expect(showSuccessToast).toHaveBeenCalledWith("Key added.");
    expect(showErrorToast).not.toHaveBeenCalled();
  });
});
