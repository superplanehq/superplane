import { useRef, type ReactElement } from "react";
import { Navigate, useLocation, useParams, useSearchParams } from "react-router-dom";

import { useCanvasConsole } from "@/hooks/useCanvasData";
import { readLastVisitedAppTab, type AppTabId } from "@/lib/lastVisitedAppTab";
import { Skeleton } from "@/ui/skeleton";

import { AppPage } from "./index";
import {
  buildAppTabSearchParams,
  resolveDefaultTab,
  urlPinsNavigation,
  urlViewFlagsToTab,
  type DefaultTabResolution,
} from "./defaultAppTab";
import { getWorkflowViewFlagsFromSearchParams } from "./viewState";

/**
 * Route-level gate that decides which tab the user should land on for an app
 * URL before AppPage mounts. Priorities:
 * 1. If the URL already pins navigation (tab-selecting `view` or a deep link
 *    like `run`/`version`/`edit`/`sidebar`/`node`/`file`), render AppPage.
 * 2. If localStorage records a last-visited tab, redirect there via Navigate.
 * 3. Otherwise consult the live console; redirect to Console if the app has
 *    panels, else render AppPage on Canvas.
 *
 * Once we hand off to AppPage for a canvas, we do not re-resolve for that
 * canvas: in-page URL changes (opening a run, switching tabs, closing a run
 * inspection back to a bare URL) must not trigger another redirect.
 */
export function AppDefaultTabGate() {
  const { appId } = useParams<{ appId: string }>();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const canvasId = appId ?? "";

  const committedCanvasIdRef = useRef<string | null>(null);
  if (committedCanvasIdRef.current !== null && committedCanvasIdRef.current !== canvasId) {
    committedCanvasIdRef.current = null;
  }
  const alreadyCommitted = canvasId !== "" && committedCanvasIdRef.current === canvasId;

  const pinned = urlPinsNavigation(searchParams);
  const storedTab = canvasId ? readLastVisitedAppTab(canvasId) : null;
  const currentUrlTab = urlViewFlagsToTab(getWorkflowViewFlagsFromSearchParams(searchParams));

  // The console query is only useful for the Console fallback (no stored tab).
  // Keep it disabled otherwise so bookmarks that pin navigation or restore a
  // stored tab do not pay for an unused read.
  const consoleQueryEnabled = !alreadyCommitted && !pinned && !!canvasId && storedTab === null;
  const liveConsoleQuery = useCanvasConsole(canvasId, undefined, consoleQueryEnabled);

  const commit = (): ReactElement => {
    committedCanvasIdRef.current = canvasId;
    return <AppPage />;
  };

  if (alreadyCommitted || !canvasId || pinned) {
    return commit();
  }

  if (storedTab !== null) {
    return resolveWithStoredTab(storedTab, currentUrlTab, searchParams, location.pathname, commit);
  }

  const resolution = resolveDefaultTab({ storedTab, liveConsoleQuery });
  return renderResolution(resolution, currentUrlTab, searchParams, location.pathname, commit);
}

function resolveWithStoredTab(
  storedTab: AppTabId,
  currentUrlTab: AppTabId | null,
  searchParams: URLSearchParams,
  pathname: string,
  commit: () => ReactElement,
): ReactElement {
  if (storedTab === currentUrlTab) {
    return commit();
  }
  return navigateToTab(storedTab, searchParams, pathname);
}

function renderResolution(
  resolution: DefaultTabResolution,
  currentUrlTab: AppTabId | null,
  searchParams: URLSearchParams,
  pathname: string,
  commit: () => ReactElement,
): ReactElement {
  if (!resolution.settled) {
    return <AppTabGateSkeleton />;
  }

  const { redirectTo } = resolution;
  if (redirectTo === null || redirectTo === currentUrlTab) {
    return commit();
  }

  return navigateToTab(redirectTo, searchParams, pathname);
}

function navigateToTab(tab: AppTabId, searchParams: URLSearchParams, pathname: string): ReactElement {
  const nextParams = buildAppTabSearchParams(tab, searchParams);
  const search = nextParams.toString();
  return <Navigate to={{ pathname, search: search ? `?${search}` : "" }} replace />;
}

/**
 * Full-viewport pulse shown while the gate waits on the live-console read.
 * Deliberately non-interactive: the user must not be able to switch tabs
 * before the redirect settles, which is exactly what removes the old
 * "user picked a tab while console was loading" race.
 */
function AppTabGateSkeleton() {
  return (
    <div
      role="status"
      aria-label="Loading app"
      aria-busy="true"
      data-testid="app-default-tab-gate-skeleton"
      className="flex h-full min-h-[400px] w-full flex-col gap-4 p-6"
    >
      <Skeleton className="h-10 w-full max-w-md" />
      <Skeleton className="h-full min-h-[240px] w-full flex-1" />
    </div>
  );
}
