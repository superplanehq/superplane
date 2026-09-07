import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";

const experimentalFeatureMocks = vi.hoisted(() => ({
  has: vi.fn((_featureId: string) => false),
  isLoading: false,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: experimentalFeatureMocks.has,
    enabledExperimentalFeatures: [],
    isLoading: experimentalFeatureMocks.isLoading,
  }),
}));

import { RequireClassicAppsSurface } from "./RequireClassicAppsSurface";

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location" data-pathname={location.pathname} />;
}

function renderGate(initialEntry: string) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route
          path="/:organizationId"
          element={
            <RequireClassicAppsSurface>
              <div data-testid="classic-surface" />
            </RequireClassicAppsSurface>
          }
        />
        <Route
          path="/:organizationId/apps/new"
          element={
            <RequireClassicAppsSurface>
              <div data-testid="classic-surface" />
            </RequireClassicAppsSurface>
          }
        />
        <Route path="/:organizationId/workspaces" element={<div data-testid="workspaces-index" />} />
      </Routes>
      <LocationProbe />
    </MemoryRouter>,
  );
}

describe("RequireClassicAppsSurface", () => {
  beforeEach(() => {
    experimentalFeatureMocks.has.mockReset();
    experimentalFeatureMocks.has.mockReturnValue(false);
    experimentalFeatureMocks.isLoading = false;
  });

  it("renders the classic surface when factories are off", () => {
    renderGate("/org-123");

    expect(screen.getByTestId("classic-surface")).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveAttribute("data-pathname", "/org-123");
  });

  it("redirects org home to /workspaces when factories are on", () => {
    experimentalFeatureMocks.has.mockImplementation((featureId) => featureId === FEATURE_FACTORIES);
    renderGate("/org-123");

    expect(screen.getByTestId("workspaces-index")).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveAttribute("data-pathname", "/org-123/workspaces");
  });

  it("redirects /apps/new to /workspaces when factories are on", () => {
    experimentalFeatureMocks.has.mockImplementation((featureId) => featureId === FEATURE_FACTORIES);
    renderGate("/org-123/apps/new");

    expect(screen.getByTestId("workspaces-index")).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveAttribute("data-pathname", "/org-123/workspaces");
  });

  it("waits while the feature flag is loading", () => {
    experimentalFeatureMocks.isLoading = true;
    renderGate("/org-123");

    expect(screen.getByTestId("classic-apps-surface-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("classic-surface")).not.toBeInTheDocument();
  });
});
