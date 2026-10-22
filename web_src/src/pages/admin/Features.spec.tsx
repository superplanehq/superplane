import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { showErrorToast } from "@/lib/toast";
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

function stubCatalog(options: { deleteResponse?: () => Promise<Response> } = {}) {
  const state = { features: [feature], deletes: [] as string[] };
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      if (init?.method !== "DELETE") return json(state.features);
      state.deletes.push(path);
      const response = await (options.deleteResponse?.() ?? Promise.resolve(new Response(null, { status: 204 })));
      if (response.ok)
        state.features = state.features.filter((item) => `/admin/api/feature-definitions/${item.key}` !== path);
      return response;
    }),
  );
  return state;
}

it("deletes a feature after confirmation and refreshes the catalog", async () => {
  const catalog = stubCatalog();
  const user = userEvent.setup();
  renderPage();
  await user.click(await screen.findByRole("button", { name: "Delete Reports" }));
  const dialog = within(screen.getByRole("dialog", { name: "Delete “Reports”?" }));
  expect(
    dialog.getByText(
      "This removes the feature from all plans and organization grants. Organizations lose access immediately. This cannot be undone.",
    ),
  ).toBeInTheDocument();
  await user.click(dialog.getByRole("button", { name: "Delete feature" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  expect(catalog.deletes).toEqual(["/admin/api/feature-definitions/reports"]);
  expect(await screen.findByText("No features. Create a feature to add it to plans.")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Create feature" })).toHaveFocus();
});

it("sends no delete request when the user keeps the feature or presses Escape", async () => {
  const catalog = stubCatalog();
  const user = userEvent.setup();
  renderPage();
  const trigger = await screen.findByRole("button", { name: "Delete Reports" });
  await user.click(trigger);
  await user.click(screen.getByRole("button", { name: "Keep feature" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await waitFor(() => expect(trigger).toHaveFocus());
  await user.click(trigger);
  await user.keyboard("{Escape}");
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await waitFor(() => expect(trigger).toHaveFocus());
  expect(catalog.deletes).toEqual([]);
  expect(screen.getByRole("button", { name: "Delete Reports" })).toBeInTheDocument();
});

it("sends one delete request and blocks dismissal while the request is pending", async () => {
  let resolveDelete!: (response: Response) => void;
  const catalog = stubCatalog({
    deleteResponse: () =>
      new Promise<Response>((resolve) => {
        resolveDelete = resolve;
      }),
  });
  const user = userEvent.setup();
  renderPage();
  await user.click(await screen.findByRole("button", { name: "Delete Reports" }));
  const dialog = screen.getByRole("dialog", { name: "Delete “Reports”?" });
  await user.click(within(dialog).getByRole("button", { name: "Delete feature" }));
  const pending = within(dialog).getByRole("button", { name: "Deleting..." });
  expect(pending).toBeDisabled();
  expect(within(dialog).getByRole("button", { name: "Keep feature" })).toBeDisabled();
  await user.keyboard("{Escape}");
  pending.click();
  expect(screen.getByRole("dialog", { name: "Delete “Reports”?" })).toBeInTheDocument();
  expect(catalog.deletes).toHaveLength(1);
  resolveDelete(new Response(null, { status: 204 }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  expect(catalog.deletes).toHaveLength(1);
  expect(await screen.findByText("No features. Create a feature to add it to plans.")).toBeInTheDocument();
});

it("shows a delete error in the dialog, allows retry, and keeps the feature until deletion succeeds", async () => {
  let attempts = 0;
  const catalog = stubCatalog({
    deleteResponse: async () =>
      ++attempts === 1
        ? new Response("Failed to update billing catalog", { status: 500 })
        : new Response(null, { status: 204 }),
  });
  const user = userEvent.setup();
  renderPage();
  await user.click(await screen.findByRole("button", { name: "Delete Reports" }));
  const dialog = within(screen.getByRole("dialog", { name: "Delete “Reports”?" }));
  await user.click(dialog.getByRole("button", { name: "Delete feature" }));
  expect(await dialog.findByRole("alert")).toHaveTextContent("Failed to update billing catalog");
  expect(dialog.getByRole("button", { name: "Keep feature" })).toBeEnabled();
  expect(dialog.getByRole("button", { name: "Delete feature" })).toBeEnabled();
  expect(catalog.deletes).toHaveLength(1);
  await user.click(dialog.getByRole("button", { name: "Delete feature" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  expect(catalog.deletes).toHaveLength(2);
  expect(await screen.findByText("No features. Create a feature to add it to plans.")).toBeInTheDocument();
});

it("restores focus to the row trigger when the user dismisses after a delete error", async () => {
  stubCatalog({ deleteResponse: async () => new Response("Failed to update billing catalog", { status: 500 }) });
  const user = userEvent.setup();
  renderPage();
  const trigger = await screen.findByRole("button", { name: "Delete Reports" });
  await user.click(trigger);
  const dialog = within(screen.getByRole("dialog", { name: "Delete “Reports”?" }));
  await user.click(dialog.getByRole("button", { name: "Delete feature" }));
  expect(await dialog.findByRole("alert")).toBeInTheDocument();
  await user.click(dialog.getByRole("button", { name: "Keep feature" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  await waitFor(() => expect(trigger).toHaveFocus());
  expect(screen.getByRole("button", { name: "Delete Reports" })).toBeInTheDocument();
});

it("shows the catalog load error after a successful delete without repeating the delete", async () => {
  let deletes = 0;
  let loads = 0;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "DELETE") {
        deletes += 1;
        return new Response(null, { status: 204 });
      }
      loads += 1;
      if (loads === 1) return json([feature]);
      if (loads === 2) return new Response("Unavailable", { status: 500 });
      return json([]);
    }),
  );
  const user = userEvent.setup();
  renderPage();
  await user.click(await screen.findByRole("button", { name: "Delete Reports" }));
  await user.click(screen.getByRole("button", { name: "Delete feature" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load features");
  expect(showErrorToast).not.toHaveBeenCalled();
  expect(deletes).toBe(1);
  await user.click(screen.getByRole("button", { name: "Try again" }));
  expect(await screen.findByText("No features. Create a feature to add it to plans.")).toBeInTheDocument();
  expect(deletes).toBe(1);
});
