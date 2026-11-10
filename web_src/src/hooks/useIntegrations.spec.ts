import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

const {
  organizationsListIntegrationResources,
  integrationsListIntegrations,
  organizationsNextIntegrationSetupStep,
  organizationsPreviousIntegrationSetupStep,
  organizationsUpdateIntegrationCapabilities,
  organizationsUpdateIntegrationProperty,
  organizationsUpdateIntegrationSecret,
  organizationsDeleteIntegration,
} = vi.hoisted(() => ({
  organizationsListIntegrationResources: vi.fn(),
  integrationsListIntegrations: vi.fn(),
  organizationsNextIntegrationSetupStep: vi.fn(),
  organizationsPreviousIntegrationSetupStep: vi.fn(),
  organizationsUpdateIntegrationCapabilities: vi.fn(),
  organizationsUpdateIntegrationProperty: vi.fn(),
  organizationsUpdateIntegrationSecret: vi.fn(),
  organizationsDeleteIntegration: vi.fn(),
}));

const { useOrganizationIdMock } = vi.hoisted(() => ({
  useOrganizationIdMock: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  organizationsListIntegrationResources,
  integrationsListIntegrations,
  organizationsNextIntegrationSetupStep,
  organizationsPreviousIntegrationSetupStep,
  organizationsUpdateIntegrationCapabilities,
  organizationsUpdateIntegrationProperty,
  organizationsUpdateIntegrationSecret,
  organizationsDeleteIntegration,
}));

vi.mock("./useOrganizationId", () => ({
  useOrganizationId: useOrganizationIdMock,
}));

import {
  integrationKeys,
  resolveRepositoryDefaultBranch,
  useAvailableIntegrations,
  useDeleteIntegration,
  useNextIntegrationSetupStep,
  usePreviousIntegrationSetupStep,
  useUpdateIntegrationCapabilities,
  useUpdateIntegrationProperty,
  useUpdateIntegrationSecret,
} from "./useIntegrations";

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

const ORGANIZATION_ID = "explicit-org";
const INTEGRATION_ID = "int-1";

function setPathname(pathname: string) {
  window.history.pushState({}, "", pathname);
}

async function expectOrganizationHeader<T>(
  useHook: () => { mutateAsync: (input: T) => Promise<unknown> },
  input: T,
  client: ReturnType<typeof vi.fn>,
) {
  const queryClient = createQueryClient();
  const { result } = renderHook(useHook, { wrapper: createWrapper(queryClient) });

  await result.current.mutateAsync(input);

  expect(client).toHaveBeenCalledWith(
    expect.objectContaining({
      path: { id: ORGANIZATION_ID, integrationId: INTEGRATION_ID },
      headers: expect.objectContaining({ "x-organization-id": ORGANIZATION_ID }),
    }),
  );
}

describe("integration mutations on /onboarding", () => {
  const mutationClients = [
    organizationsNextIntegrationSetupStep,
    organizationsPreviousIntegrationSetupStep,
    organizationsUpdateIntegrationCapabilities,
    organizationsUpdateIntegrationProperty,
    organizationsUpdateIntegrationSecret,
    organizationsDeleteIntegration,
  ];

  beforeEach(() => {
    setPathname("/onboarding");
    for (const client of mutationClients) {
      client.mockReset();
      client.mockResolvedValue({ data: {} });
    }
  });

  afterEach(() => {
    setPathname("/");
  });

  it.each([
    {
      name: "next setup step",
      run: () =>
        expectOrganizationHeader(
          () => useNextIntegrationSetupStep(ORGANIZATION_ID),
          { integrationId: INTEGRATION_ID },
          organizationsNextIntegrationSetupStep,
        ),
    },
    {
      name: "previous setup step",
      run: () =>
        expectOrganizationHeader(
          () => usePreviousIntegrationSetupStep(ORGANIZATION_ID),
          { integrationId: INTEGRATION_ID },
          organizationsPreviousIntegrationSetupStep,
        ),
    },
    {
      name: "capability update",
      run: () =>
        expectOrganizationHeader(
          () => useUpdateIntegrationCapabilities(ORGANIZATION_ID, INTEGRATION_ID),
          [],
          organizationsUpdateIntegrationCapabilities,
        ),
    },
    {
      name: "property update",
      run: () =>
        expectOrganizationHeader(
          () => useUpdateIntegrationProperty(ORGANIZATION_ID, INTEGRATION_ID),
          { propertyName: "host", value: "acme.ghe.com" },
          organizationsUpdateIntegrationProperty,
        ),
    },
    {
      name: "secret update",
      run: () =>
        expectOrganizationHeader(
          () => useUpdateIntegrationSecret(ORGANIZATION_ID, INTEGRATION_ID),
          { secretName: "token", value: "secret-value" },
          organizationsUpdateIntegrationSecret,
        ),
    },
    {
      name: "delete",
      run: () =>
        expectOrganizationHeader(
          () => useDeleteIntegration(ORGANIZATION_ID, INTEGRATION_ID),
          { integrationName: "github" },
          organizationsDeleteIntegration,
        ),
    },
  ])("sends the hook organization id on the $name", async ({ run }) => {
    await run();
  });
});
