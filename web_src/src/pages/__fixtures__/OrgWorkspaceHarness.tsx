import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { MemoryRouter, Outlet, Route, Routes } from "react-router-dom";

import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { AccountProvider } from "@/contexts/AccountProvider";
import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { AppPage } from "@/pages/app";
import { canvasAppIds, type CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";
import { HomePage } from "@/pages/home";
import { homePageIds, type HomePageFixture } from "@/pages/home/__fixtures__/handlers";
import { NewAppPage } from "@/pages/home/NewAppPage";
import { TooltipProvider } from "@/ui/tooltip";

import { createOrgWorkspaceFixtureFetch } from "./createOrgWorkspaceFixtureFetch";

interface FixtureFetchState {
  original: typeof fetch;
  delegate: typeof fetch | null;
}

const FIXTURE_FETCH_KEY = "__orgWorkspaceFixtureFetch";

/**
 * Installs a permanent `window.fetch` wrapper (once) that delegates to the
 * currently active fixture fetch, or to the real network when no story is
 * mounted. Same race-safe delegate pattern as the page harnesses.
 */
function fixtureFetchState(): FixtureFetchState {
  const holder = window as unknown as Record<string, FixtureFetchState | undefined>;
  let state = holder[FIXTURE_FETCH_KEY];
  if (!state) {
    const created: FixtureFetchState = { original: window.fetch.bind(window), delegate: null };
    holder[FIXTURE_FETCH_KEY] = created;
    window.fetch = ((input: RequestInfo | URL, init?: RequestInit) =>
      (created.delegate ?? created.original)(input, init)) as typeof fetch;
    state = created;
  }
  return state;
}

export interface OrgWorkspaceHarnessProps {
  /** Where to land when the story mounts. */
  startAt?: "home" | "app";
  /** Path under the org when `startAt` is `home`, e.g. `apps/new`. */
  pathSuffix?: string;
  /** Query string for the app route (without leading `?`). */
  appQuery?: string;
  /**
   * When true, opens the canvas agent sidebar via localStorage before mount.
   * Always written (true/false) so story switches do not leak open state.
   */
  openAgentSidebar?: boolean;
  homeFixture?: HomePageFixture;
  appFixture?: CanvasAppFixture;
}

/**
 * Shared Storybook shell for org home + app editor so the real React Router
 * links work: logo/Homepage → home, Software Factory card → live canvas.
 */
export function OrgWorkspaceHarness({
  startAt = "home",
  pathSuffix = "",
  appQuery = "",
  openAgentSidebar = false,
  homeFixture,
  appFixture,
}: OrgWorkspaceHarnessProps) {
  const orgId = homeFixture?.organizationId ?? appFixture?.organizationId ?? homePageIds.organizationId;
  const canvasId = appFixture?.canvasId ?? canvasAppIds.canvasId;

  const [fixtureFetch] = useState(() => {
    // Persist before AppPage reads the preference in useState initializers.
    writeCanvasAgentSidebarOpen(canvasId, openAgentSidebar);
    const state = fixtureFetchState();
    const impl = createOrgWorkspaceFixtureFetch(state.original, { homeFixture, appFixture });
    state.delegate = impl;
    return impl;
  });

  useEffect(() => {
    writeCanvasAgentSidebarOpen(canvasId, openAgentSidebar);
    const state = fixtureFetchState();
    if (state.delegate === null) {
      state.delegate = fixtureFetch;
    }
    return () => {
      if (state.delegate === fixtureFetch) {
        state.delegate = null;
      }
    };
  }, [canvasId, fixtureFetch, openAgentSidebar]);

  const homePath = pathSuffix ? `/${orgId}/${pathSuffix}` : `/${orgId}`;
  const appPath = `/${orgId}/apps/${canvasId}${appQuery ? `?${appQuery}` : ""}`;
  const initialPath = startAt === "app" ? appPath : homePath;

  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={150}>
        <div className="h-dvh w-full overflow-auto">
          <MemoryRouter initialEntries={[initialPath]}>
            <AccountProvider>
              <Routes>
                <Route
                  path=":organizationId"
                  element={
                    <PermissionsProvider>
                      <Outlet />
                    </PermissionsProvider>
                  }
                >
                  <Route index element={<HomePage />} />
                  <Route path="apps/new" element={<NewAppPage />} />
                  <Route path="apps/:appId" element={<AppPage />} />
                </Route>
              </Routes>
            </AccountProvider>
          </MemoryRouter>
        </div>
      </TooltipProvider>
    </QueryClientProvider>
  );
}
