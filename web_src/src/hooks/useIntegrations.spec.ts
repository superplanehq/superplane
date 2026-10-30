import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { organizationsListIntegrationResources, integrationsListIntegrations } = vi.hoisted(() => ({
  organizationsListIntegrationResources: vi.fn(),
  integrationsListIntegrations: vi.fn(),
}));

const { useOrganizationIdMock } = vi.hoisted(() => ({
  useOrganizationIdMock: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  organizationsListIntegrationResources,
  integrationsListIntegrations,
}));

vi.mock("./useOrganizationId", () => ({
  useOrganizationId: useOrganizationIdMock,
}));

import { integrationKeys, resolveRepositoryDefaultBranch, useAvailableIntegrations } from "./useIntegrations";

describe("resolveRepositoryDefaultBranch", () => {
  beforeEach(() => {
    organizationsListIntegrationResources.mockReset();
  });

  it("returns the branch reported by the integration", async () => {
    organizationsListIntegrationResources.mockResolvedValue({
      data: { resources: [{ type: "default_branch", name: "staging", id: "staging" }] },
    });

    const branch = await resolveRepositoryDefaultBranch("org-1", "int-1", "acme/app");

    expect(branch).toBe("staging");
    expect(organizationsListIntegrationResources).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { id: "org-1", integrationId: "int-1" },
        query: { type: "default_branch", repository: "acme/app" },
      }),
    );
  });

  it("returns an empty branch when the organization id is missing", async () => {
    const branch = await resolveRepositoryDefaultBranch("", "int-1", "acme/app");

    expect(branch).toBe("");
    expect(organizationsListIntegrationResources).not.toHaveBeenCalled();
  });

  it("returns an empty branch when the integration id is missing", async () => {
    const branch = await resolveRepositoryDefaultBranch("org-1", "", "acme/app");

    expect(branch).toBe("");
    expect(organizationsListIntegrationResources).not.toHaveBeenCalled();
  });

  it("returns an empty branch when the repository is missing", async () => {
    const branch = await resolveRepositoryDefaultBranch("org-1", "int-1", "");

    expect(branch).toBe("");
    expect(organizationsListIntegrationResources).not.toHaveBeenCalled();
  });

  it("returns an empty branch when the response has no resources", async () => {
    organizationsListIntegrationResources.mockResolvedValue({ data: { resources: [] } });

    const branch = await resolveRepositoryDefaultBranch("org-1", "int-1", "acme/app");

    expect(branch).toBe("");
  });

  it("rejects when the branch request fails", async () => {
    organizationsListIntegrationResources.mockRejectedValue(new Error("github unavailable"));

    await expect(resolveRepositoryDefaultBranch("org-1", "int-1", "acme/app")).rejects.toThrow("github unavailable");
  });
});

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useAvailableIntegrations", () => {
  beforeEach(() => {
    integrationsListIntegrations.mockReset();
    integrationsListIntegrations.mockResolvedValue({
      data: { integrations: [], githubAppConfigured: true },
    });
    useOrganizationIdMock.mockReturnValue("org-from-route");
  });

  it("keys the catalog by the route organization when organizationId is omitted", async () => {
    const queryClient = createQueryClient();

    const { result } = renderHook(() => useAvailableIntegrations(), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryData(integrationKeys.available("org-from-route"))).toEqual(
      expect.objectContaining({ githubAppConfigured: true }),
    );
    expect(queryClient.getQueryData(integrationKeys.available(""))).toBeUndefined();
    expect(integrationsListIntegrations).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({ "x-organization-id": "org-from-route" }),
      }),
    );
  });

  it("keys the catalog by the explicit organizationId", async () => {
    const queryClient = createQueryClient();

    const { result } = renderHook(() => useAvailableIntegrations({ organizationId: "explicit-org" }), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(queryClient.getQueryData(integrationKeys.available("explicit-org"))).toEqual(
      expect.objectContaining({ githubAppConfigured: true }),
    );
    expect(queryClient.getQueryData(integrationKeys.available("org-from-route"))).toBeUndefined();
    expect(integrationsListIntegrations).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({ "x-organization-id": "explicit-org" }),
      }),
    );
  });
});
