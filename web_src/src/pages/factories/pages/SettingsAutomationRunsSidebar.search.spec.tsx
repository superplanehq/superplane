import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import type { CanvasesListRunsData } from "@/api-client";

const { canvasesListRuns } = vi.hoisted(() => ({ canvasesListRuns: vi.fn() }));
vi.mock("@/api-client/sdk.gen", () => ({ canvasesListRuns }));

import { SettingsAutomationRunsSidebar } from "./SettingsAutomationRunsSidebar";

afterEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
});

it("searches titles and triggers on the server without paging through nonmatching runs", async () => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  canvasesListRuns.mockImplementation(async ({ query }: { query: CanvasesListRunsData["query"] }) => ({
    data:
      query?.search || query?.triggerIds?.length
        ? { runs: [], totalCount: 0, hasNextPage: false }
        : {
            runs: [{ id: "run-1", rootEvent: { nodeId: "trigger-a", customName: "Deploy API" } }],
            totalCount: 1,
            hasNextPage: false,
          },
  }));

  const { unmount } = render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <SettingsAutomationRunsSidebar
          canvasId="canvas-1"
          selectedRunId={null}
          onSelectRun={() => {}}
          workflowNodes={[
            { id: "trigger-a", name: "GitLab MRs", type: "TYPE_TRIGGER", component: "gitlab.onMergeRequest" },
          ]}
        />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  await screen.findByText("Deploy API");
  fireEvent.change(screen.getByRole("textbox", { name: "Search runs" }), { target: { value: "missing" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Search runs" }), { target: { value: "missing title" } });
  await waitFor(() =>
    expect(canvasesListRuns).toHaveBeenLastCalledWith(
      expect.objectContaining({ query: { limit: 25, search: "missing title" } }),
    ),
  );
  await screen.findByText("No runs match your filters");
  expect(canvasesListRuns).toHaveBeenCalledTimes(2);

  fireEvent.click(screen.getByRole("button", { name: "Filter runs" }));
  fireEvent.click(within(screen.getByRole("dialog")).getByText("GitLab MRs"));
  await waitFor(() =>
    expect(canvasesListRuns).toHaveBeenLastCalledWith(
      expect.objectContaining({ query: { limit: 25, search: "missing title", triggerIds: ["trigger-a"] } }),
    ),
  );
  expect(canvasesListRuns).toHaveBeenCalledTimes(3);
  unmount();
  queryClient.clear();
});
