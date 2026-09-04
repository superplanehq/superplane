import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Mocking the console query avoids standing up React Query plumbing while
// giving each test full control over the async fallback state.
type ConsoleQueryLike = {
  isSuccess: boolean;
  isError: boolean;
  data: { panels: object[] } | undefined;
};

let mockConsoleQuery: ConsoleQueryLike;

function consoleLoaded(panels: object[]): ConsoleQueryLike {
  return { isSuccess: true, isError: false, data: { panels } };
}

const consoleLoading: ConsoleQueryLike = { isSuccess: false, isError: false, data: undefined };
const consoleErrored: ConsoleQueryLike = { isSuccess: false, isError: true, data: undefined };

type CanvasQueryLike = {
  data: { metadata?: { factoryId?: string } } | undefined;
  isLoading: boolean;
};

let mockCanvasQuery: CanvasQueryLike = { data: undefined, isLoading: false };

const featureMocks = vi.hoisted(() => ({
  factoriesEnabled: false,
  featureLoading: false,
  factories: [] as Array<{ id?: string; key?: string }>,
  factoriesLoading: false,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvasConsole: () => mockConsoleQuery,
  useCanvas: () => mockCanvasQuery,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (featureId: string) => featureId === "factories" && featureMocks.factoriesEnabled,
    enabledExperimentalFeatures: [],
    isLoading: featureMocks.featureLoading,
  }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => ({
    data: featureMocks.factories,
    isLoading: featureMocks.factoriesLoading,
  }),
}));

// AppPage pulls in the entire canvas surface; substitute a marker so the gate
// spec stays focused on routing decisions rather than page rendering.
vi.mock("./index", () => ({
  AppPage: () => <div data-testid="app-page" />,
}));

import { AppDefaultTabGate } from "./AppDefaultTabGate";
import { recordLastVisitedAppTab } from "@/lib/lastVisitedAppTab";

function renderGate({ initialEntry }: { initialEntry: string }) {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/:organizationId/apps/:appId" element={<AppDefaultTabGate />} />
        <Route path="/apps/:appId" element={<AppDefaultTabGate />} />
        <Route path="/:organizationId" element={<div data-testid="org-home" />} />
        <Route path="/:organizationId/workspaces" element={<div data-testid="workspaces-index" />} />
        <Route path="/:organizationId/workspaces/:factoryKey/apps/:appId" element={<div data-testid="factory-app" />} />
      </Routes>
      <LocationProbe />
    </MemoryRouter>,
  );
}

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location" data-search={location.search} data-pathname={location.pathname} />;
}

function getLocation() {
  const el = screen.getByTestId("location");
  return {
    pathname: el.getAttribute("data-pathname") ?? "",
    search: el.getAttribute("data-search") ?? "",
  };
}

beforeEach(() => {
  window.localStorage.clear();
  mockConsoleQuery = consoleLoaded([]);
  mockCanvasQuery = { data: undefined, isLoading: false };
  featureMocks.factoriesEnabled = false;
  featureMocks.featureLoading = false;
  featureMocks.factories = [];
  featureMocks.factoriesLoading = false;
});

describe("AppDefaultTabGate — pinned URLs", () => {
  it("renders AppPage immediately when the URL already picks a tab", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    renderGate({ initialEntry: "/apps/canvas-1?view=memory" });

    expect(screen.getByTestId("app-page")).toBeInTheDocument();
    expect(getLocation().search).toBe("?view=memory");
  });

  it("renders AppPage without redirecting when the URL deep-links to a run", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    renderGate({ initialEntry: "/apps/canvas-1?run=run-1" });

    expect(screen.getByTestId("app-page")).toBeInTheDocument();
    expect(getLocation().search).toBe("?run=run-1");
  });

  it.each(["version=v1", "edit=1", "node=n1&sidebar=1", "file=components%2Fapp.yaml"])(
    "renders AppPage without redirecting for deep link %s",
    (query) => {
      recordLastVisitedAppTab("canvas-1", "console");
      renderGate({ initialEntry: `/apps/canvas-1?${query}` });

      expect(screen.getByTestId("app-page")).toBeInTheDocument();
      expect(getLocation().search).toBe(`?${query}`);
    },
  );

  it("renders AppPage for the legacy Console alias without touching the URL", () => {
    renderGate({ initialEntry: "/apps/canvas-1?view=dashboard" });

    expect(screen.getByTestId("app-page")).toBeInTheDocument();
    expect(getLocation().search).toBe("?view=dashboard");
  });
});

describe("AppDefaultTabGate — stored-tab redirect", () => {
  it("navigates to the stored tab before AppPage renders", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    renderGate({ initialEntry: "/apps/canvas-1" });

    expect(getLocation().search).toBe("?view=console");
    expect(screen.getByTestId("app-page")).toBeInTheDocument();
  });

  it("does not navigate when the stored tab already matches the current URL tab", () => {
    recordLastVisitedAppTab("canvas-1", "canvas");
    renderGate({ initialEntry: "/apps/canvas-1" });

    expect(getLocation().search).toBe("");
    expect(screen.getByTestId("app-page")).toBeInTheDocument();
  });

  it("does not consult the console query when a stored tab is present", () => {
    // If the console query were consulted it would be loading — the gate would
    // show the skeleton and never render AppPage.
    mockConsoleQuery = consoleLoading;
    recordLastVisitedAppTab("canvas-1", "memory");
    renderGate({ initialEntry: "/apps/canvas-1" });

    expect(getLocation().search).toBe("?view=memory");
    expect(screen.getByTestId("app-page")).toBeInTheDocument();
  });

  it("preserves the legacy view value on redirect since it does not pin navigation", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    renderGate({ initialEntry: "/apps/canvas-1?view=runs" });

    // Legacy `view=runs` is not tab-selecting; the stored redirect still fires
    // and useWorkflowViewSearchParams cleans up the leftover legacy value.
    expect(getLocation().search).toBe("?view=console");
  });
});

describe("AppDefaultTabGate — factory apps", () => {
  it("sends factory-owned apps to org home when factories are off", () => {
    mockCanvasQuery = { data: { metadata: { factoryId: "factory-1" } }, isLoading: false };
    renderGate({ initialEntry: "/org-1/apps/canvas-1?view=console" });

    expect(getLocation().pathname).toBe("/org-1");
    expect(screen.getByTestId("org-home")).toBeInTheDocument();
  });

  it("sends leftover classic apps to /workspaces when factories are on", () => {
    featureMocks.factoriesEnabled = true;
    mockCanvasQuery = { data: { metadata: {} }, isLoading: false };
    renderGate({ initialEntry: "/org-1/apps/canvas-1" });

    expect(getLocation().pathname).toBe("/org-1/workspaces");
    expect(screen.getByTestId("workspaces-index")).toBeInTheDocument();
  });

  it("sends factory-owned apps to the workspace editor when factories are on", () => {
    featureMocks.factoriesEnabled = true;
    featureMocks.factories = [{ id: "factory-1", key: "RF" }];
    mockCanvasQuery = { data: { metadata: { factoryId: "factory-1" } }, isLoading: false };
    renderGate({ initialEntry: "/org-1/apps/canvas-1" });

    expect(getLocation().pathname).toBe("/org-1/workspaces/RF/apps/canvas-1");
    expect(getLocation().search).toBe("?configure=1&agent=1");
    expect(screen.getByTestId("factory-app")).toBeInTheDocument();
  });

  it("keeps a run query on the workspace app URL", () => {
    featureMocks.factoriesEnabled = true;
    featureMocks.factories = [{ id: "factory-1", key: "RF" }];
    mockCanvasQuery = { data: { metadata: { factoryId: "factory-1" } }, isLoading: false };
    renderGate({ initialEntry: "/org-1/apps/canvas-1?run=run-9" });

    expect(getLocation().pathname).toBe("/org-1/workspaces/RF/apps/canvas-1");
    expect(getLocation().search).toBe("?run=run-9");
    expect(screen.getByTestId("factory-app")).toBeInTheDocument();
  });
});

describe("AppDefaultTabGate — first-visit console fallback", () => {
  it("shows the skeleton while the console query is loading", () => {
    mockConsoleQuery = consoleLoading;
    renderGate({ initialEntry: "/apps/canvas-1" });

    expect(screen.getByTestId("app-default-tab-gate-skeleton")).toBeInTheDocument();
    expect(screen.queryByTestId("app-page")).toBeNull();
  });

  it("redirects to Console once the query reports panels", () => {
    mockConsoleQuery = consoleLoaded([{ id: "p1" }]);
    renderGate({ initialEntry: "/apps/canvas-1" });

    expect(getLocation().search).toBe("?view=console");
    expect(screen.getByTestId("app-page")).toBeInTheDocument();
  });

  it("stays on Canvas when the app has no panels", () => {
    mockConsoleQuery = consoleLoaded([]);
    renderGate({ initialEntry: "/apps/canvas-1" });

    expect(getLocation().search).toBe("");
    expect(screen.getByTestId("app-page")).toBeInTheDocument();
  });

  it("stays on Canvas when the console query errors", () => {
    mockConsoleQuery = consoleErrored;
    renderGate({ initialEntry: "/apps/canvas-1" });

    expect(getLocation().search).toBe("");
    expect(screen.getByTestId("app-page")).toBeInTheDocument();
  });
});
