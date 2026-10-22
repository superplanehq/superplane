import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { Features } from "./Features";

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn() }));
afterEach(() => vi.unstubAllGlobals());
const feature = { key: "reports", label: "Reports", gated: false };
const plan = { id: "plan-1", key: "custom", name: "Custom" };
const rows = [
  { id: "grant", name: "Grant org", slug: "grant-org", enabled_grant: true, plan: null },
  { id: "plan", name: "Plan org", slug: "plan-org", enabled_grant: false, plan },
  { id: "both", name: "Both org", slug: "both-org", enabled_grant: true, plan },
];
const json = (value: unknown) => new Response(JSON.stringify(value));
const renderPage = () =>
  render(
    <MemoryRouter>
      <Features />
    </MemoryRouter>,
  );

it("shows independent access sources and organization links, searches and pages, and restores focus on Escape", async () => {
  const paths: string[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path.endsWith("feature-definitions")) return json([feature]);
      paths.push(path);
      return json({ items: rows, total: 51, limit: 50, offset: path.includes("offset=50") ? 50 : 0 });
    }),
  );
  const user = userEvent.setup();
  renderPage();
  const trigger = await screen.findByRole("button", { name: "View organizations" });
  await user.click(trigger);
  const drawer = within(screen.getByRole("dialog", { name: "Reports" }));
  expect(drawer.getByText("reports", { exact: true })).toBeInTheDocument();
  expect(drawer.getByText(/Every organization has access/)).toBeInTheDocument();
  for (const row of rows) {
    const link = await drawer.findByRole("link", { name: row.name });
    expect(link).toHaveAttribute("href", `/admin/organizations/${row.id}`);
    const cells = within(link.closest("tr")!);
    expect(cells.queryByText("Grant") !== null).toBe(row.enabled_grant);
    expect(cells.queryByText("Plan", { exact: true }) !== null).toBe(row.plan !== null);
    if (row.plan) expect(cells.getByText("Custom")).toBeInTheDocument();
  }
  expect(drawer.getAllByRole("link", { name: "Both org" })).toHaveLength(1);
  await user.click(drawer.getByRole("button", { name: "Next" }));
  await waitFor(() => expect(paths.at(-1)).toContain("offset=50"));
  await user.type(drawer.getByRole("searchbox", { name: "Search organizations" }), "Grant");
  await waitFor(() => expect(paths.at(-1)).toContain("search=Grant&limit=50&offset=0"));
  await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await waitFor(() => expect(trigger).toHaveFocus());
});

it("shows drawer loading, retry, and an empty search result", async () => {
  let resolveRequest!: (response: Response) => void;
  let attempt = 0;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).endsWith("feature-definitions")) return json([{ ...feature, gated: true }]);
      if (++attempt === 1)
        return new Promise<Response>((resolve) => {
          resolveRequest = resolve;
        });
      return json({ items: [], total: 0, limit: 50, offset: 0 });
    }),
  );
  const user = userEvent.setup();
  renderPage();
  await user.click(await screen.findByRole("button", { name: "View organizations" }));
  expect(screen.getByRole("status")).toHaveTextContent("Loading organizations");
  resolveRequest(new Response("Service unavailable", { status: 500 }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load organizations");
  await user.click(screen.getByRole("button", { name: "Try again" }));
  expect(await screen.findByText("No organizations have an enabled grant or plan inclusion.")).toBeInTheDocument();
  expect(screen.queryByText(/Every organization has access/)).not.toBeInTheDocument();
  await user.type(screen.getByRole("searchbox"), "missing");
  expect(await screen.findByText("No matching organizations. Try a different search.")).toBeInTheDocument();
});

it("edits feature access on the Features page and returns focus to Edit", async () => {
  const updates: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PUT") {
        updates.push(JSON.parse(String(init.body)));
        return json({});
      }
      return json([feature]);
    }),
  );
  const user = userEvent.setup();
  renderPage();
  const edit = await screen.findByRole("button", { name: "Edit Reports" });
  await user.click(edit);
  expect(screen.getByLabelText("Feature key")).toBeDisabled();
  await user.click(screen.getByRole("checkbox", { name: "Require plan selection or an organization grant" }));
  await user.click(screen.getByRole("button", { name: "Save feature" }));
  await waitFor(() => expect(updates).toEqual([{ ...feature, gated: true }]));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await waitFor(() => expect(screen.getByRole("button", { name: "Edit Reports" })).toHaveFocus());
});

it("loads the catalog and retries a failed request", async () => {
  let resolveRequest!: (response: Response) => void;
  vi.stubGlobal(
    "fetch",
    vi.fn(
      () =>
        new Promise<Response>((resolve) => {
          resolveRequest = resolve;
        }),
    ),
  );
  const user = userEvent.setup();
  renderPage();
  expect(screen.getByRole("status")).toHaveTextContent("Loading features");
  resolveRequest(new Response("Unavailable", { status: 500 }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load features");
  await user.click(screen.getByRole("button", { name: "Try again" }));
  resolveRequest(json([]));
  expect(await screen.findByText("No features. Create a feature to add it to plans.")).toBeInTheDocument();
});
