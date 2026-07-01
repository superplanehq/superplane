import type { OrganizationsOrganization } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { useOrganizationIdMock } = vi.hoisted(() => ({
  useOrganizationIdMock: vi.fn(),
}));

vi.mock("./useOrganizationId", () => ({
  useOrganizationId: useOrganizationIdMock,
}));

import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import {
  experimentalFeaturesKeys,
  type ExperimentalFeature,
  type ExperimentalFeaturesRegistry,
} from "@/hooks/useExperimentalFeatures";
import { organizationKeys } from "@/hooks/useOrganizationData";

const ORG_ID = "org-1";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function makeFeature(overrides: Partial<ExperimentalFeature> & { id: string }): ExperimentalFeature {
  return {
    label: overrides.id,
    description: "",
    released: false,
    ...overrides,
  };
}

function seedQueries(
  queryClient: QueryClient,
  {
    organization,
    registry,
    orgId = ORG_ID,
  }: {
    organization?: OrganizationsOrganization | null;
    registry?: ExperimentalFeaturesRegistry;
    orgId?: string;
  },
) {
  if (organization !== undefined) {
    queryClient.setQueryData(organizationKeys.details(orgId), organization);
  }
  if (registry !== undefined) {
    queryClient.setQueryData(experimentalFeaturesKeys.registry(), registry);
  }
}

beforeEach(() => {
  useOrganizationIdMock.mockReturnValue(ORG_ID);
});

describe("useExperimentalFeature", () => {
  it("returns has function and enabledExperimentalFeatures array", () => {
    const queryClient = createQueryClient();

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(Object.keys(result.current).sort()).toEqual(["enabledExperimentalFeatures", "has"]);
    expect(result.current.has).toEqual(expect.any(Function));
    expect(result.current.enabledExperimentalFeatures).toEqual(expect.any(Array));
  });

  it("returns false when the feature is not in the registry, even if the org has opted in", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: {
        spec: { enabledExperimentalFeatures: ["ghost"] },
      } as OrganizationsOrganization,
      registry: { features: [makeFeature({ id: "alpha" })] },
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.has("ghost")).toBe(false);
  });

  it("returns true when the feature is marked released, regardless of org opt-in", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: {
        spec: { enabledExperimentalFeatures: [] },
      } as OrganizationsOrganization,
      registry: {
        features: [makeFeature({ id: "alpha", released: true })],
      },
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.has("alpha")).toBe(true);
  });

  it("returns true when the feature exists and the organization has opted in", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: {
        spec: { enabledExperimentalFeatures: ["alpha"] },
      } as OrganizationsOrganization,
      registry: {
        features: [makeFeature({ id: "alpha" })],
      },
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.has("alpha")).toBe(true);
  });

  it("returns false when the feature exists but the organization has not opted in", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: {
        spec: { enabledExperimentalFeatures: [] },
      } as OrganizationsOrganization,
      registry: {
        features: [makeFeature({ id: "alpha" })],
      },
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.has("alpha")).toBe(false);
  });

  it("returns false when enabledExperimentalFeatures is undefined on the organization", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: { spec: {} } as OrganizationsOrganization,
      registry: {
        features: [makeFeature({ id: "alpha" })],
      },
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.has("alpha")).toBe(false);
  });

  it("returns false when no organization id is available", () => {
    useOrganizationIdMock.mockReturnValue(null);
    const queryClient = createQueryClient();

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.has("alpha")).toBe(false);
  });

  it("returns false while the registry has not loaded yet", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: {
        spec: { enabledExperimentalFeatures: ["alpha"] },
      } as OrganizationsOrganization,
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.has("alpha")).toBe(false);
  });

  it("lists released features and features the organization opted into", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: {
        spec: { enabledExperimentalFeatures: ["beta"] },
      } as OrganizationsOrganization,
      registry: {
        features: [
          makeFeature({ id: "alpha", released: true }),
          makeFeature({ id: "beta" }),
          makeFeature({ id: "gamma" }),
        ],
      },
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.enabledExperimentalFeatures).toEqual(["alpha", "beta"]);
  });

  it("returns an empty list while the registry has not loaded yet", () => {
    const queryClient = createQueryClient();
    seedQueries(queryClient, {
      organization: {
        spec: { enabledExperimentalFeatures: ["alpha"] },
      } as OrganizationsOrganization,
    });

    const { result } = renderHook(() => useExperimentalFeature(), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current.enabledExperimentalFeatures).toEqual([]);
  });
});
