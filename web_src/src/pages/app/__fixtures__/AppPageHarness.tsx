import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { AppPage } from "..";
import { canvasAppIds, createFixtureFetch, type CanvasAppFixture } from "./handlers";

interface FixtureFetchState {
  original: typeof fetch;
  delegate: typeof fetch | null;
}

// The holder lives on `window` (not in module scope) so Storybook HMR module
// reloads reuse the existing wrapper instead of wrapping it again.
const FIXTURE_FETCH_KEY = "__appPageFixtureFetch";

/**
 * Installs a permanent `window.fetch` wrapper (once) that delegates to the
 * currently active fixture fetch, or to the real network when no story is
 * mounted. Swapping a delegate instead of swapping `window.fetch` itself
 * means story transitions can never race: the new story's install and the
 * old story's cleanup each only touch their own delegate slot.
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

interface AppPageHarnessProps {
  /** Query string appended to the AppPage route (without the leading `?`). */
  query?: string;
  /**
   * Fixture to serve for this story. Defaults to the Clean Code Assessment
   * capture used by the original `LiveCanvas`/`RunInspection` stories.
   */
  fixture?: CanvasAppFixture;
}

/**
 * Mounts `AppPage` against an in-process fixture backend. Every story that
 * exercises the full app orchestrator shares this harness so the fetch
 * override, memory router, and React Query wiring stay in one place.
 *
 * The fixture fetch is activated *synchronously during render* — before
 * `AppPage` mounts and its React Query hooks fire — so no request ever
 * reaches the network. A lazy `useState` initializer is the earliest safe
 * hook point; a `useEffect` would run after the child queries had already
 * started. Each harness instance registers its own delegate, so switching
 * stories always serves the incoming story's fixture even while the outgoing
 * story's cleanup hasn't run yet.
 */
export function AppPageHarness({ query = "", fixture }: AppPageHarnessProps) {
  const [fixtureFetch] = useState(() => {
    const state = fixtureFetchState();
    const impl = createFixtureFetch(state.original, fixture);
    state.delegate = impl;
    return impl;
  });

  useEffect(() => {
    const state = fixtureFetchState();
    // Re-claim the delegate if the slot is free so StrictMode's dev-only
    // unmount/remount cycle doesn't leave it cleared by our own cleanup.
    // (The initial claim happens in the useState initializer above.)
    if (state.delegate === null) {
      state.delegate = fixtureFetch;
    }
    return () => {
      // Only clear the delegate if it is still ours; a newer story may have
      // already installed its own fixture by the time this cleanup runs.
      if (state.delegate === fixtureFetch) {
        state.delegate = null;
      }
    };
  }, [fixtureFetch]);

  const orgId = fixture?.organizationId ?? canvasAppIds.organizationId;
  const canvasId = fixture?.canvasId ?? canvasAppIds.canvasId;
  const path = `/${orgId}/apps/${canvasId}${query ? `?${query}` : ""}`;

  // A fresh client per story keeps them isolated from one another and from the
  // shared preview client (which caches with staleTime: Infinity).
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <TooltipProvider delayDuration={150}>
          {/* AppPage's root is `h-full`, so it needs an ancestor with an explicit
              height (the real app gets this from AppRouter's `h-dvh` wrapper). */}
          <div className="h-dvh w-full">
            <MemoryRouter initialEntries={[path]}>
              <Routes>
                <Route
                  path=":organizationId/apps/:appId"
                  element={
                    <PermissionsProvider>
                      <AppPage />
                    </PermissionsProvider>
                  }
                />
              </Routes>
            </MemoryRouter>
          </div>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
