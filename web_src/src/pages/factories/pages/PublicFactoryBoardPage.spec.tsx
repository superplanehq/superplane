import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { PublicFactoryBoardPage } from "./PublicFactoryBoardPage";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: { id: "user-1", name: "Ada Lovelace", installation_admin: false },
  }),
}));

vi.mock("@/posthog", () => ({
  posthog: { reset: vi.fn() },
}));

const boardPath = "/demo/workspaces/newwo/lines/line-1";

function renderBoard(signedIn: boolean, accountName?: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ThemeProvider>
        <MemoryRouter initialEntries={[boardPath]}>
          <Routes>
            <Route
              path="/:organizationId/workspaces/:factoryKey/lines/:lineId"
              element={<PublicFactoryBoardPage signedIn={signedIn} accountName={accountName} />}
            />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

describe("PublicFactoryBoardPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("stays on the board when a guest receives 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("Not found", { status: 404 })),
    );

    renderBoard(false);

    expect(await screen.findByText("This board is not available.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute(
      "href",
      expect.stringContaining("/login?redirect="),
    );
    expect(window.location.pathname).not.toContain("/login");
  });

  it("lists column automations without links and shows a public view badge", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({
              workspaceName: "Instabot",
              lineName: "implement",
              showClarity: false,
              showConfidence: false,
              columns: [
                {
                  key: "backlog",
                  title: "Backlog",
                  automations: [
                    {
                      id: "intake-0",
                      kind: "intake",
                      name: "GitHub issues",
                      catalogId: "github-issues",
                      icon: "github",
                      health: "healthy",
                    },
                  ],
                  cards: [
                    {
                      id: "backlog-0",
                      title: "refine agents.md",
                      createdAt: "2026-09-29T12:00:00Z",
                      state: "STATE_DRAFT",
                      origin: { url: "https://github.com/acme/instabot/issues/12", label: "acme#12" },
                      assignee: { name: "Ada Lovelace", avatarUrl: "https://example.com/ada.png" },
                      confidence: 4,
                    },
                  ],
                },
                {
                  key: "done",
                  title: "Done",
                  automations: [
                    {
                      id: "closure",
                      kind: "pr-closure",
                      name: "PR Closure",
                      catalogId: "pr-closure",
                      icon: "github",
                      health: "healthy",
                    },
                  ],
                  cards: [],
                },
              ],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          ),
      ),
    );

    renderBoard(false);

    expect(await screen.findByTestId("workspace-page-header-title")).toHaveTextContent("Instabot");
    expect(screen.getByTestId("workspace-page-header-title")).not.toHaveTextContent("Implement");
    expect(screen.getByTestId("public-board-badge")).toHaveTextContent("Public view");
    expect(screen.queryByTestId("work-orders-scope")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-trigger")).toBeInTheDocument();
    expect(screen.getByTestId("factories-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("factories-nav-board")).toBeInTheDocument();
    expect(screen.queryByTestId("factories-nav-velocity")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-sidebar-user-menu")).not.toBeInTheDocument();
    expect(screen.getByTestId("factories-workspace-switch")).toHaveAccessibleName("Instabot");
    const intake = screen.getByText("Listens to GitHub issues");
    const closure = screen.getByText("Closes tasks when pull requests merge");
    expect(intake.closest("button")).toBeNull();
    expect(closure.closest("button")).toBeNull();
    expect(intake.closest("a")).toBeNull();
    expect(screen.getByRole("heading", { name: "refine agents.md" })).toBeInTheDocument();
    const card = screen.getByTestId("work-order-card-backlog-0");
    expect(within(card).getByTestId("work-order-row-assignees-backlog-0")).toHaveAttribute("title", "Ada Lovelace");
    expect(screen.getByTestId("work-order-card-source-backlog-0")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open refine agents.md" })).not.toBeInTheDocument();
  });

  it("opens a read-only task modal from a public card and closes it", async () => {
    const card = {
      number: 7,
      id: "backlog-0",
      title: "Public task",
      createdAt: "2026-10-15T12:00:00Z",
      state: "STATE_DRAFT",
    };
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const task = String(input).endsWith("/tasks/7");
      return new Response(
        JSON.stringify(
          task
            ? {
                card,
                description: "Task description",
                column: "Backlog",
                totalTokens: 1234,
                comments: [{ body: "Ready for review", createdAt: card.createdAt }],
                runs: [{ name: "Implementation", status: "finished", result: "passed", durationSeconds: 12 }],
                pullRequests: [{ number: 42, state: "open", url: "https://github.com/acme/app/pull/42" }],
              }
            : {
                workspaceName: "Public workspace",
                workspaceKey: "PUB",
                columns: [{ key: "backlog", title: "Backlog", cards: [card], automations: [] }],
              },
        ),
        { headers: { "Content-Type": "application/json" } },
      );
    });
    vi.stubGlobal("fetch", fetchMock);
    renderBoard(false);
    fireEvent.click(await screen.findByRole("button", { name: "Open Public task" }));
    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("Task description")).toBeInTheDocument();
    expect(within(dialog).getByText("Ready for review")).toBeInTheDocument();
    expect(within(dialog).getByText("Implementation").closest("button")).toBeNull();
    expect(within(dialog).getByRole("link", { name: "Review PR #42" })).toHaveAttribute(
      "href",
      "https://github.com/acme/app/pull/42",
    );
    expect(within(dialog).queryByRole("textbox")).not.toBeInTheDocument();
    expect(within(dialog).queryByText(/Spend|Artifacts|Logs/)).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.some(([url]) => String(url).endsWith("/lines/line-1/tasks/7"))).toBe(true);
    expect(fetchMock.mock.calls.some(([url]) => /\/api\/v1\/factories\//.test(String(url)))).toBe(false);
    fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows the signed-in user on the rail", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(JSON.stringify({ workspaceName: "Instabot", lineName: "implement", columns: [] }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
      ),
    );

    renderBoard(true, "Ada Lovelace");

    expect(await screen.findByTestId("factories-sidebar-user-menu-trigger")).toHaveAccessibleName(/Ada Lovelace/);
    expect(screen.queryByText("Switch workspace")).not.toBeInTheDocument();
  });

  it("shows permission denied when a member receives 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("Not found", { status: 404 })),
    );

    renderBoard(true);

    expect(await screen.findByText("You do not have permission to open this workspace.")).toBeInTheDocument();
  });

  it("offers home when a signed-in board request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("error", { status: 500 })),
    );

    renderBoard(true, "Ada Lovelace");

    expect(await screen.findByRole("link", { name: "Go to home" })).toHaveAttribute("href", "/");
    expect(screen.queryByRole("link", { name: "Sign in" })).not.toBeInTheDocument();
    expect(window.location.pathname).not.toContain("/login");
  });
});
