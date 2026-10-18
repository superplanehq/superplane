import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";

import OrganizationDetail from "./OrganizationDetail";

const ORG_ID = "org-1";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Admin" } }),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        MemoryRouter,
        { initialEntries: [`/admin/organizations/${ORG_ID}`] },
        createElement(Routes, null, createElement(Route, { path: "/admin/organizations/:orgId", element: children })),
      ),
    );
  return render(<OrganizationDetail />, { wrapper });
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function organizationOverview(memberCount: number) {
  return {
    id: ORG_ID,
    name: "Acme",
    slug: "acme",
    description: "Builds widgets",
    canvas_count: 2,
    task_count: 4,
    done_task_count: 6,
    member_count: memberCount,
    workspace_count: 7,
    repository_count: 8,
    created_at: "2024-01-15T12:00:00Z",
    updated_at: "2024-02-20T12:00:00Z",
  };
}

describe("OrganizationDetail overview", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("refreshes the member count when Overview opens again", async () => {
    const user = userEvent.setup();
    let memberCount = 3;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === `/admin/api/organizations/${ORG_ID}`) {
          return jsonResponse(organizationOverview(memberCount));
        }
        if (url.startsWith(`/admin/api/organizations/${ORG_ID}/users`)) {
          return jsonResponse({ items: [], total: 0 });
        }
        return new Response("not found", { status: 404 });
      }),
    );

    renderPage();

    const overview = await screen.findByRole("tabpanel", { name: "Overview" });
    expect(within(overview).getByText("Members").nextElementSibling).toHaveTextContent("3");
    const detailCalls = () =>
      vi.mocked(fetch).mock.calls.filter(([input]) => String(input) === `/admin/api/organizations/${ORG_ID}`).length;
    expect(detailCalls()).toBe(1);

    await user.click(screen.getByRole("tab", { name: "Users" }));
    memberCount = 9;

    await user.click(screen.getByRole("tab", { name: "Overview" }));

    const refreshed = await screen.findByRole("tabpanel", { name: "Overview" });
    expect(await within(refreshed).findByText("9")).toBeInTheDocument();
    expect(within(refreshed).getByText("Members").nextElementSibling).toHaveTextContent("9");
    expect(detailCalls()).toBe(2);
  });
});
