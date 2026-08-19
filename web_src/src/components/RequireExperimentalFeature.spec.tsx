import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it } from "vitest";
import type { OrganizationsOrganization } from "@/api-client";
import { experimentalFeaturesKeys, type ExperimentalFeaturesRegistry } from "@/hooks/useExperimentalFeatures";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { RequireExperimentalFeature } from "./RequireExperimentalFeature";

function renderGate(
  initialEntry: string,
  queries: { organization?: OrganizationsOrganization; registry?: ExperimentalFeaturesRegistry },
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  if (queries.organization !== undefined) {
    queryClient.setQueryData(organizationKeys.details("org-123"), queries.organization);
  }
  if (queries.registry !== undefined) {
    queryClient.setQueryData(experimentalFeaturesKeys.registry(), queries.registry);
  }

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/:organizationId" element={<div>Home</div>} />
          <Route
            path="/:organizationId/factories"
            element={
              <RequireExperimentalFeature featureId="factories">
                <div>Factories page</div>
              </RequireExperimentalFeature>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("RequireExperimentalFeature", () => {
  it("shows a loading state while feature access is unresolved", () => {
    renderGate("/org-123/factories", {});

    expect(screen.getByText("Loading...")).toBeInTheDocument();
    expect(screen.queryByText("Factories page")).not.toBeInTheDocument();
    expect(screen.queryByText("Home")).not.toBeInTheDocument();
  });

  it("redirects home when the feature is disabled after loading", () => {
    renderGate("/org-123/factories", {
      organization: {
        spec: { enabledExperimentalFeatures: [] },
      } as OrganizationsOrganization,
      registry: {
        features: [{ id: "factories", label: "Factories", description: "", released: false }],
      },
    });

    expect(screen.getByText("Home")).toBeInTheDocument();
    expect(screen.queryByText("Factories page")).not.toBeInTheDocument();
  });

  it("renders children when the feature is enabled", () => {
    renderGate("/org-123/factories", {
      organization: {
        spec: { enabledExperimentalFeatures: ["factories"] },
      } as OrganizationsOrganization,
      registry: {
        features: [{ id: "factories", label: "Factories", description: "", released: false }],
      },
    });

    expect(screen.getByText("Factories page")).toBeInTheDocument();
  });
});
