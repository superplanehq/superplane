import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { AppPage } from "..";
import { canvasAppIds, createFixtureFetch, type CanvasAppFixture } from "./handlers";

type PatchedFetch = typeof fetch & { __fixtureFetch?: boolean };

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
 * The fixture fetch is installed *synchronously during render* — before
 * `AppPage` mounts and its React Query hooks fire — so no request ever
 * reaches the network. A lazy `useState` initializer is the earliest safe
 * hook point; a `useEffect` would run after the child queries had already
 * started. Nested `AppPageHarness` renders (a story wrapper mounting another)
 * see the sentinel `__fixtureFetch` flag and reuse the outer install rather
 * than double-wrapping.
 */
export function AppPageHarness({ query = "", fixture }: AppPageHarnessProps) {
  const originalFetch = useRef<typeof fetch | null>(null);

  useState(() => {
    if (!(window.fetch as PatchedFetch).__fixtureFetch) {
      originalFetch.current = window.fetch.bind(window);
      const patched = createFixtureFetch(originalFetch.current, fixture) as PatchedFetch;
      patched.__fixtureFetch = true;
      window.fetch = patched;
    }
    return null;
  });

  useEffect(() => {
    return () => {
      if (originalFetch.current) {
        window.fetch = originalFetch.current;
      }
    };
  }, []);

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
