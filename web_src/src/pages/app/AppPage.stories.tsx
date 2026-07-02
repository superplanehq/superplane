import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { AppPage } from "./index";
import { canvasAppIds, createFixtureFetch } from "./__fixtures__/handlers";

/**
 * Mounts the real `AppPage` orchestrator against an in-process fixture backend
 * seeded from a live canvas capture (see `__fixtures__/canvasAppResponses.json`).
 *
 * Networking is faked by overriding `window.fetch` (see `createFixtureFetch`)
 * rather than MSW: MSW relies on a Service Worker, which is silently disabled in
 * non-secure contexts (opening Storybook via a LAN IP instead of `localhost`),
 * causing every request to escape to the live API. The fetch override has no
 * such dependency, so the graph, runs sidebar, versions, and run-inspection
 * detail pane render deterministic fake data however Storybook is opened.
 */
const meta = {
  title: "Pages/AppPage",
  component: AppPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof AppPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const appPath = (query: string) =>
  `/${canvasAppIds.organizationId}/apps/${canvasAppIds.canvasId}${query ? `?${query}` : ""}`;

type PatchedFetch = typeof fetch & { __fixtureFetch?: boolean };

function AppPageHarness({ query }: { query: string }) {
  const originalFetch = useRef<typeof fetch | null>(null);

  // Install the fixture fetch *synchronously during render* — before AppPage
  // mounts and its React Query hooks fire — so no request ever reaches the
  // network. A lazy useState initializer is the earliest safe hook point; a
  // useEffect would run after the child queries have already started.
  useState(() => {
    if (!(window.fetch as PatchedFetch).__fixtureFetch) {
      originalFetch.current = window.fetch.bind(window);
      const patched = createFixtureFetch(originalFetch.current) as PatchedFetch;
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

  // A fresh client per story keeps them isolated from one another and from the
  // shared preview client (which caches with staleTime: Infinity).
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={150}>
        {/* AppPage's root is `h-full`, so it needs an ancestor with an explicit
            height (the real app gets this from AppRouter's `h-dvh` wrapper). */}
        <div className="h-dvh w-full">
          <MemoryRouter initialEntries={[appPath(query)]}>
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
    </QueryClientProvider>
  );
}

/** Live canvas view: the ReactFlow graph plus the runs history sidebar. */
export const LiveCanvas: Story = {
  render: () => <AppPageHarness query="" />,
};

/**
 * Run inspection: a finished (passed) run is selected and the bottom
 * `RunNodeDetailPane` is opened on the `post-assessment` node, showing that
 * node's execution output for the run.
 */
export const RunInspection: Story = {
  render: () => <AppPageHarness query={`run=${canvasAppIds.publishedRunId}&sidebar=1&node=post-assessment`} />,
};
