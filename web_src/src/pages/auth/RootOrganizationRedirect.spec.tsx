import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RootOrganizationRedirect } from "./RootOrganizationRedirect";

const accountState = vi.hoisted(() => ({ account: { id: "account-1" } as { id: string } | null }));
const organizationsState = vi.hoisted(() => ({
  data: [{ id: "org-uuid-1", slug: "acme", name: "Acme" }] as Array<{ id: string; slug?: string; name: string }>,
  isLoading: false,
  isError: false,
}));
const lastLocationState = vi.hoisted(() => ({
  data: null as string | null,
  isLoading: false,
  isError: false,
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => accountState,
}));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => organizationsState,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => false, enabledExperimentalFeatures: [], isLoading: false }),
}));

vi.mock("@/hooks/useLastLocation", () => ({
  useLastLocation: () => lastLocationState,
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location">{`${location.pathname}${location.search}`}</div>;
}

function renderRedirect() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route path="/" element={<RootOrganizationRedirect />} />
          <Route path="*" element={<LocationDisplay />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("RootOrganizationRedirect", () => {
  beforeEach(() => {
    accountState.account = { id: "account-1" };
    organizationsState.data = [{ id: "org-uuid-1", slug: "acme", name: "Acme" }];
    organizationsState.isLoading = false;
    organizationsState.isError = false;
    lastLocationState.data = null;
    lastLocationState.isLoading = false;
    lastLocationState.isError = false;
    window.localStorage.clear();
  });

  it("resumes the saved screen when the backend has one", async () => {
    lastLocationState.data = "/acme/apps/deploy?run=42&node=approve-1";

    renderRedirect();

    await waitFor(() => {
      expect(screen.getByTestId("location")).toHaveTextContent("/acme/apps/deploy?run=42&node=approve-1");
    });
  });

  it("falls back to the organization home page when there is nothing saved", async () => {
    renderRedirect();

    await waitFor(() => {
      expect(screen.getByTestId("location")).toHaveTextContent("/acme");
    });
  });

  it("falls back to local storage when the backend request fails", async () => {
    lastLocationState.isError = true;
    window.localStorage.setItem(
      "superplane:last-visited-location",
      JSON.stringify({ "account-1:acme": "/acme/apps/deploy?run=7" }),
    );

    renderRedirect();

    await waitFor(() => {
      expect(screen.getByTestId("location")).toHaveTextContent("/acme/apps/deploy?run=7");
    });
  });

  it("shows a loading state while the last location is still resolving", () => {
    lastLocationState.isLoading = true;

    renderRedirect();

    expect(screen.getByText("Loading...")).toBeInTheDocument();
    expect(screen.queryByTestId("location")).not.toBeInTheDocument();
  });
});
