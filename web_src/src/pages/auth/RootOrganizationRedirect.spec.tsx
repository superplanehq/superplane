import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { LAST_VISITED_FACTORY_STORAGE_KEY } from "../factories/lib/lastVisitedFactory";
import { RootOrganizationRedirect } from "./RootOrganizationRedirect";

const accountState = vi.hoisted(() => ({ account: { id: "account-1" } as { id: string } | null }));
const organizationsState = vi.hoisted(() => ({
  data: [{ id: "org-uuid-1", slug: "acme", name: "Acme" }] as Array<{
    id: string;
    slug?: string;
    name: string;
    lastLocationPath?: string;
    lastLocationUpdatedAt?: string;
  }>,
  isLoading: false,
  isError: false,
}));
const lastLocationState = vi.hoisted(() => ({
  data: null as string | null,
  isLoading: false,
  isError: false,
}));
const experimentalState = vi.hoisted(() => ({
  has: (_feature?: string): boolean => false,
  enabledExperimentalFeatures: [] as string[],
  isLoading: false,
}));
const workspacesState = vi.hoisted(() => ({
  data: [] as Array<{ id?: string; key?: string; onboarding?: { completedAt?: string } }>,
  isLoading: false,
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => accountState,
}));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => organizationsState,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => experimentalState,
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => workspacesState,
}));

vi.mock("@/hooks/useLastLocation", () => ({
  useLastLocation: () => lastLocationState,
}));

function LocationDisplay() {
  const location = useLocation();
  return <div data-testid="location">{`${location.pathname}${location.search}`}</div>;
}

function expectPath(path: string) {
  expect(screen.getByTestId("location").textContent).toBe(path);
}

function rememberLastWorkspace(factoryId: string, organizationRoute = "acme") {
  window.localStorage.setItem(
    LAST_VISITED_FACTORY_STORAGE_KEY,
    JSON.stringify({ "account-1": { [organizationRoute]: factoryId } }),
  );
}

const finishedPay = {
  id: "factory-pay",
  key: "PAY",
  onboarding: { completedAt: "2026-09-01T00:00:00.000Z" },
};
const finishedShip = {
  id: "factory-ship",
  key: "SHIP",
  onboarding: { completedAt: "2026-09-02T00:00:00.000Z" },
};
const unfinishedPay = { id: "factory-pay", key: "PAY", onboarding: {} };
const unfinishedOther = { id: "factory-other", key: "OTHER", onboarding: {} };

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
    experimentalState.has = () => false;
    experimentalState.isLoading = false;
    workspacesState.data = [];
    workspacesState.isLoading = false;
    window.localStorage.clear();
  });

  it("resumes the saved screen when the backend has one", async () => {
    lastLocationState.data = "/acme/apps/deploy?run=42&node=approve-1";

    renderRedirect();

    await waitFor(() => {
      expect(screen.getByTestId("location")).toHaveTextContent("/acme/apps/deploy?run=42&node=approve-1");
    });
  });

  it("sends an account with no organization to onboarding", async () => {
    organizationsState.data = [];

    renderRedirect();

    await waitFor(() => {
      expect(screen.getByTestId("location")).toHaveTextContent("/onboarding");
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

  it("picks the organization with the newest saved screen when last visited is gone", async () => {
    organizationsState.data = [
      { id: "org-old", slug: "puppies-inc", name: "Old", lastLocationUpdatedAt: "2026-02-01T00:00:00.000Z" },
      {
        id: "org-live",
        slug: "acme",
        name: "Acme",
        lastLocationPath: "/acme/apps/deploy?run=9",
        lastLocationUpdatedAt: "2026-09-01T00:00:00.000Z",
      },
    ];
    window.localStorage.setItem("superplane:last-visited-organization", JSON.stringify({ "account-1": "gone-slug" }));

    renderRedirect();

    await waitFor(() => {
      expect(screen.getByTestId("location")).toHaveTextContent("/acme");
    });
  });

  it("shows an error when organizations fail to load", () => {
    organizationsState.isError = true;

    renderRedirect();

    expect(
      screen.getByText("We could not load your organizations. Refresh the page and try again."),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("location")).not.toBeInTheDocument();
  });

  it("opens setup for the only unfinished workspace when factories are on", async () => {
    experimentalState.has = () => true;
    workspacesState.data = [unfinishedPay];

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/pay/setup");
    });
  });

  it("opens the last visited workspace and drops the saved task query", async () => {
    experimentalState.has = () => true;
    lastLocationState.data = "/acme/workspaces/pay/task/12?tab=notes";
    workspacesState.data = [finishedPay, finishedShip];
    rememberLastWorkspace("factory-ship");

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/ship");
    });
  });

  it("opens the workspace from a saved velocity path when no last workspace is stored", async () => {
    experimentalState.has = () => true;
    lastLocationState.data = "/acme/workspaces/Pay/velocity?range=30d";
    workspacesState.data = [finishedPay, finishedShip];

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/pay");
    });
  });

  it("recovers the workspace key from local storage when the saved screen request fails", async () => {
    experimentalState.has = () => true;
    lastLocationState.isError = true;
    window.localStorage.setItem(
      "superplane:last-visited-location",
      JSON.stringify({ "account-1:acme": "/acme/workspaces/ship/velocity" }),
    );
    workspacesState.data = [finishedPay, finishedShip];

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/ship");
    });
  });

  it("skips an unfinished last workspace when another finished workspace exists", async () => {
    experimentalState.has = () => true;
    workspacesState.data = [unfinishedPay, finishedShip];
    rememberLastWorkspace("factory-pay");

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/ship");
    });
  });

  it("opens setup for the chosen unfinished workspace, not the first unfinished one", async () => {
    experimentalState.has = () => true;
    workspacesState.data = [unfinishedOther, unfinishedPay];
    rememberLastWorkspace("factory-pay");

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/pay/setup");
    });
  });

  it("does not reopen a saved app screen when factories are on", async () => {
    experimentalState.has = () => true;
    lastLocationState.data = "/acme/apps/deploy?run=42";
    workspacesState.data = [finishedPay];

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/pay");
    });
  });

  it("ignores a saved new-workspace path and opens a finished workspace", async () => {
    experimentalState.has = () => true;
    lastLocationState.data = "/acme/workspaces/new";
    workspacesState.data = [finishedShip];

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/ship");
    });
  });

  it("recovers the workspace key from the organization saved path when the server screen is empty", async () => {
    experimentalState.has = () => true;
    organizationsState.data = [
      {
        id: "org-uuid-1",
        slug: "acme",
        name: "Acme",
        lastLocationPath: "/acme/workspaces/ship/tasks",
      },
    ];
    workspacesState.data = [finishedPay, finishedShip];

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces/ship");
    });
  });

  it("opens the workspace list when the organization has no workspace", async () => {
    experimentalState.has = () => true;
    lastLocationState.data = "/acme/workspaces/pay/task/4";
    workspacesState.data = [];

    renderRedirect();

    await waitFor(() => {
      expectPath("/acme/workspaces");
    });
  });
});
