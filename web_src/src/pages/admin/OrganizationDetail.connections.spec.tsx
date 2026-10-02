import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";

import OrganizationDetail from "./OrganizationDetail";

const ORG_ID = "org-1";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Admin" } }),
}));

function organization() {
  return {
    id: ORG_ID,
    name: "Acme",
    slug: "acme",
    description: "Builds widgets",
    canvas_count: 2,
    task_count: 4,
    done_task_count: 6,
    member_count: 3,
    created_at: "2024-01-15T12:00:00Z",
    updated_at: "2024-02-20T12:00:00Z",
  };
}

function connection(
  name: string,
  id: string,
  state = "ready",
  stateDescription = "",
  details: Record<string, string> = {},
) {
  return {
    id,
    app_name: "sentry",
    installation_name: name,
    state,
    state_description: stateDescription,
    details,
    created_at: "2024-01-15T12:00:00Z",
    updated_at: "2024-02-20T12:00:00Z",
  };
}

function page(items: unknown[], total: number, offset: number) {
  return { items, total, limit: 50, offset };
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
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

function integrationsUrl(input: RequestInfo | URL) {
  const url = String(input);
  if (!url.startsWith(`/admin/api/organizations/${ORG_ID}/integrations`)) {
    return null;
  }
  return new URL(url, "http://localhost");
}

describe("OrganizationDetail connections", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === `/admin/api/organizations/${ORG_ID}`) {
          return jsonResponse(organization());
        }
        const integrations = integrationsUrl(input);
        if (integrations) {
          return jsonResponse(
            page(
              [
                connection("acme-sentry", "integration-1", "error", "Sentry is not sending issue events.", {
                  installation_uuid: "install-uuid-1",
                  external_organization: "acme-sentry-org",
                }),
              ],
              1,
              0,
            ),
          );
        }
        return new Response("not found", { status: 404 });
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows connection status and details after a tab click", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Connections" }));

    const panel = screen.getByRole("tabpanel", { name: "Connections" });
    expect(await within(panel).findByText("acme-sentry")).toBeInTheDocument();
    expect(within(panel).getByText("error")).toBeInTheDocument();
    expect(within(panel).getByText("Sentry is not sending issue events.")).toBeInTheDocument();
    expect(within(panel).getByText("install-uuid-1")).toBeInTheDocument();
    expect(within(panel).getByText("acme-sentry-org")).toBeInTheDocument();

    expect(screen.getByRole("textbox", { name: "Search connections" })).toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: "Search connections" }), "acme");
    await waitFor(() => {
      const urls = vi.mocked(fetch).mock.calls.map(([input]) => String(input));
      expect(urls.some((url) => url.includes("search=acme") && url.includes("limit=50"))).toBe(true);
    });
  });

  it("loads the next page of connections", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input) === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organization());
      }
      const integrations = integrationsUrl(input);
      if (integrations) {
        const offset = integrations.searchParams.get("offset");
        const name = offset === "50" ? "second-sentry" : "acme-sentry";
        const id = offset === "50" ? "integration-2" : "integration-1";
        return jsonResponse(page([connection(name, id)], 51, Number(offset ?? 0)));
      }
      return new Response("not found", { status: 404 });
    });

    renderPage();
    await user.click(await screen.findByRole("tab", { name: "Connections" }));
    expect(await screen.findByText("acme-sentry")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByText("second-sentry")).toBeInTheDocument();
    expect(screen.queryByText("acme-sentry")).not.toBeInTheDocument();
  });

  it("hides the current page while the next page loads", async () => {
    const user = userEvent.setup();
    let releaseNextPage: (response: Response) => void = () => {};
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input) === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organization());
      }
      const integrations = integrationsUrl(input);
      if (integrations) {
        if (integrations.searchParams.get("offset") === "50") {
          return new Promise((resolve) => {
            releaseNextPage = resolve;
          });
        }
        return jsonResponse(page([connection("acme-sentry", "integration-1")], 51, 0));
      }
      return new Response("not found", { status: 404 });
    });

    renderPage();
    await user.click(await screen.findByRole("tab", { name: "Connections" }));
    expect(await screen.findByText("acme-sentry")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByText("Loading...")).toBeInTheDocument();
    expect(screen.queryByText("acme-sentry")).not.toBeInTheDocument();

    releaseNextPage(jsonResponse(page([connection("second-sentry", "integration-2")], 51, 50)));
    expect(await screen.findByText("second-sentry")).toBeInTheDocument();
  });

  it("returns to an earlier page when a later page is empty", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input) === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organization());
      }
      const integrations = integrationsUrl(input);
      if (integrations) {
        if (integrations.searchParams.get("offset") === "50") {
          return jsonResponse(page([], 50, 50));
        }
        return jsonResponse(page([connection("acme-sentry", "integration-1")], 51, 0));
      }
      return new Response("not found", { status: 404 });
    });

    renderPage();
    await user.click(await screen.findByRole("tab", { name: "Connections" }));
    expect(await screen.findByText("acme-sentry")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => {
      const urls = vi.mocked(fetch).mock.calls.map(([input]) => String(input));
      expect(urls.filter((url) => url.includes("offset=50")).length).toBeGreaterThan(0);
      expect(urls.filter((url) => url.includes("offset=0")).length).toBeGreaterThan(1);
    });
    expect(screen.getByText("acme-sentry")).toBeInTheDocument();
    expect(screen.queryByText("This organization has no connections.")).not.toBeInTheDocument();
  });

  it("keeps a newer search when an older empty page finishes reading", async () => {
    const user = userEvent.setup();
    let releaseEmptyPage = () => {};

    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input) === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organization());
      }
      const integrations = integrationsUrl(input);
      if (integrations) {
        if (integrations.searchParams.get("offset") === "50") {
          const response = new Response(null, { status: 200 });
          response.json = () =>
            new Promise((resolve) => {
              releaseEmptyPage = () => resolve(page([], 50, 50));
            });
          return response;
        }
        if (integrations.searchParams.get("search") === "found") {
          return jsonResponse(page([connection("found-sentry", "integration-3")], 1, 0));
        }
        return jsonResponse(page([connection("acme-sentry", "integration-1")], 51, 0));
      }
      return new Response("not found", { status: 404 });
    });

    renderPage();
    await user.click(await screen.findByRole("tab", { name: "Connections" }));
    expect(await screen.findByText("acme-sentry")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByText("Loading...")).toBeInTheDocument();

    await user.type(screen.getByRole("textbox", { name: "Search connections" }), "found");
    expect(await screen.findByText("found-sentry")).toBeInTheDocument();

    await act(async () => {
      releaseEmptyPage();
    });

    expect(screen.getByText("found-sentry")).toBeInTheDocument();
    expect(screen.queryByText("acme-sentry")).not.toBeInTheDocument();
  });

  it("clears a search that matches no connections", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input) === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organization());
      }
      const integrations = integrationsUrl(input);
      if (integrations) {
        if (integrations.searchParams.get("search")) {
          return jsonResponse(page([], 0, 0));
        }
        return jsonResponse(page([connection("acme-sentry", "integration-1")], 1, 0));
      }
      return new Response("not found", { status: 404 });
    });

    renderPage();
    await user.click(await screen.findByRole("tab", { name: "Connections" }));
    expect(await screen.findByText("acme-sentry")).toBeInTheDocument();

    await user.type(screen.getByRole("textbox", { name: "Search connections" }), "none");
    expect(await screen.findByText("No connections match this search.")).toBeInTheDocument();
    expect(screen.getByText("Try a different name, or clear the search.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear search" }));
    expect(await screen.findByText("acme-sentry")).toBeInTheDocument();
    expect(screen.queryByText("No connections match this search.")).not.toBeInTheDocument();
  });

  it("explains how to add connections when the organization has none", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input) === `/admin/api/organizations/${ORG_ID}`) {
        return jsonResponse(organization());
      }
      if (integrationsUrl(input)) {
        return jsonResponse(page([], 0, 0));
      }
      return new Response("not found", { status: 404 });
    });

    renderPage();
    await user.click(await screen.findByRole("tab", { name: "Connections" }));

    expect(await screen.findByText("This organization has no connections.")).toBeInTheDocument();
    expect(screen.getByText("Members add connections on the organization integrations page.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open integrations" })).toHaveAttribute(
      "href",
      "/org-1/organization/integrations",
    );
  });
});
