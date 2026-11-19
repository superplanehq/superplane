import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, useSearchParams } from "react-router";

import { showErrorToast } from "@/lib/toast";
import { TooltipProvider } from "@/ui/tooltip";

import { SsoSettingsScreen } from "./FactorySettingsSsoPage";

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn(), showSuccessToast: vi.fn() }));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: null }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: undefined }),
}));

vi.mock("@/hooks/useSso", () => ({
  useSsoConnection: () => ({
    isLoading: false,
    error: null,
    data: {
      exists: true,
      connection: {
        id: "conn",
        status: "STATUS_DRAFT",
        displayName: "Okta",
        preset: "PRESET_OKTA",
        issuer: "https://example.okta.com",
        clientId: "client",
        clientSecretSet: true,
      },
    },
  }),
  useSsoDomains: () => ({ data: [], isLoading: false, error: null }),
  useSsoScimTokens: () => ({ data: [], isLoading: false, error: null }),
  useSaveSsoConnection: () => ({ isPending: false, mutate: vi.fn() }),
  useActivateSsoConnection: () => ({ isPending: false, mutate: vi.fn() }),
  useDisableSsoConnection: () => ({ isPending: false, mutate: vi.fn() }),
  useDeleteSsoConnection: () => ({ isPending: false, mutate: vi.fn() }),
  useCreateSsoDomain: () => ({ isPending: false, mutate: vi.fn() }),
  useVerifySsoDomain: () => ({ isPending: false, mutate: vi.fn() }),
  useDeleteSsoDomain: () => ({ isPending: false, mutate: vi.fn() }),
  useCreateSsoScimToken: () => ({ isPending: false, mutate: vi.fn() }),
  useRevokeSsoScimToken: () => ({ isPending: false, mutate: vi.fn() }),
}));

function QueryProbe() {
  const [params] = useSearchParams();
  return <div data-testid="query">{params.toString()}</div>;
}

const discoveryFailure = "SuperPlane could not read the identity provider configuration. Check the issuer URL.";

describe("SSO connection test result", () => {
  beforeEach(() => {
    vi.mocked(showErrorToast).mockClear();
  });

  it("shows a discovery failure and removes the query", async () => {
    render(
      <MemoryRouter initialEntries={["/org/workspaces/line/settings/organization/sso?sso_test=failed&reason=begin"]}>
        <TooltipProvider>
          <SsoSettingsScreen organizationId="org" />
          <QueryProbe />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith(discoveryFailure);
      expect(screen.getByTestId("query").textContent ?? "").not.toContain("sso_test");
      expect(screen.getByTestId("query").textContent ?? "").not.toContain("reason");
    });
    expect(screen.getByText(discoveryFailure)).toBeInTheDocument();
  });
});
