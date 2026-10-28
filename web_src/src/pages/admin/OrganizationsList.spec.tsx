import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";

import OrganizationsList from "./OrganizationsList";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("OrganizationsList", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          items: [
            {
              id: "org-1",
              name: "Acme",
              canvas_count: 2,
              task_count: 5,
              done_task_count: 6,
              member_count: 3,
              workspace_count: 7,
              repository_count: 8,
              created_at: "2024-01-15T12:00:00Z",
            },
          ],
          total: 1,
        }),
      ),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows automations and tasks and hides description", async () => {
    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("link", { name: "Acme" })).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Search by name or ID...")).toBeInTheDocument();
    expect(screen.getAllByRole("link")).toHaveLength(1);
    expect(screen.getByRole("row", { name: /Acme/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Automations" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Tasks$/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Done Tasks" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Workspaces" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Repositories" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Canvases" })).not.toBeInTheDocument();
    expect(screen.queryByRole("columnheader", { name: "Description" })).not.toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("6")).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
    expect(screen.getByText("8")).toBeInTheDocument();
    expect(screen.getByText("$0.00")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Reset one automation" })).not.toBeInTheDocument();
  });

  it("opens the organization when any cell in the row is clicked", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter initialEntries={["/admin"]}>
        <Routes>
          <Route path="/admin" element={<OrganizationsList />} />
          <Route path="/admin/organizations/:orgId" element={<OrganizationPath />} />
        </Routes>
      </MemoryRouter>,
    );

    await user.click(await screen.findByText("6"));

    expect(await screen.findByText("/admin/organizations/org-1")).toBeInTheDocument();
  });

  it("shows pinned organizations and counts every search match", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          items: [{ ...organization("org-1", "Acme"), plan_key: "trial", plan_name: "Trial" }],
          pinned: [{ ...organization("pinned-org", "Pinned Customer"), plan_key: "custom", plan_name: "Custom Team" }],
          total: 1,
          match_total: 4,
        }),
      ),
    );

    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("heading", { name: "Pinned" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Pinned Customer" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Unpin organization" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Pin organization" })).toBeInTheDocument();
    expect(screen.getByText("4 organizations across this installation")).toBeInTheDocument();
    const pinnedSection = screen.getByRole("region", { name: "Pinned" });
    expect(within(pinnedSection).getByText("Custom Team")).toBeInTheDocument();
    expect(within(pinnedSection).getByRole("columnheader", { name: "Plan" })).not.toHaveAttribute("aria-sort");
    expect(within(pinnedSection).queryByRole("button", { name: "Plan" })).not.toBeInTheDocument();
    expect(within(screen.getByRole("row", { name: /Acme/ })).getByText("Trial")).toBeInTheDocument();
    expect(within(pinnedSection).getByRole("columnheader", { name: "Name" })).not.toHaveAttribute("aria-sort");
    expect(within(pinnedSection).queryByRole("button", { name: "Name" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Name" })).toBeInTheDocument();
  });

  it("does not open the organization when the pin control is clicked", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/pin") && init?.method === "PUT") {
        return jsonResponse({ pinned: true });
      }
      return jsonResponse({
        items: [organization("org-1", "Acme")],
        pinned: [],
        total: 1,
        match_total: 1,
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter initialEntries={["/admin"]}>
        <Routes>
          <Route path="/admin" element={<OrganizationsList />} />
          <Route path="/admin/organizations/:orgId" element={<OrganizationPath />} />
        </Routes>
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Pin organization" }));

    expect(screen.queryByText("/admin/organizations/org-1")).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/admin/api/organizations/org-1/pin", {
      method: "PUT",
      credentials: "include",
    });
  });

  it("moves to the previous page when a pin empties the last page", async () => {
    const user = userEvent.setup();
    let pinned = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/pin") && init?.method === "PUT") {
        pinned = true;
        return jsonResponse({ pinned: true });
      }
      const offset = Number(new URL(url, "http://localhost").searchParams.get("offset") ?? "0");
      if (offset > 0) {
        return jsonResponse({
          items: pinned ? [] : [organization("org-2", "Last Org")],
          pinned: pinned ? [organization("org-2", "Last Org")] : [],
          total: pinned ? 50 : 51,
          match_total: 51,
        });
      }
      return jsonResponse({
        items: [organization("org-1", "Acme")],
        pinned: pinned ? [organization("org-2", "Last Org")] : [],
        total: pinned ? 50 : 51,
        match_total: 51,
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Next" }));
    await user.click(await screen.findByRole("button", { name: "Pin organization" }));

    expect(await screen.findByRole("heading", { name: "Pinned" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Last Org" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Acme" })).toBeInTheDocument();
    expect(screen.queryByText("Showing 51–51 of 51")).not.toBeInTheDocument();
    expect(
      fetchMock.mock.calls.some(([input]) => String(input).includes("offset=0") && String(input).includes("limit=50")),
    ).toBe(true);
  });

  it("does not replace the current search when a pin request finishes later", async () => {
    const user = userEvent.setup();
    let releasePin: (response: Response) => void = () => {};
    const pinResponse = new Promise<Response>((resolve) => {
      releasePin = resolve;
    });
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/pin") && init?.method === "PUT") {
        return pinResponse;
      }
      const search = new URL(url, "http://localhost").searchParams.get("search") ?? "";
      if (search === "Beta") {
        return jsonResponse({
          items: [organization("org-2", "Beta Org")],
          pinned: [],
          total: 1,
          match_total: 1,
        });
      }
      return jsonResponse({
        items: [organization("org-1", "Acme")],
        pinned: [],
        total: 1,
        match_total: 1,
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Pin organization" }));
    await user.type(screen.getByPlaceholderText("Search by name or ID..."), "Beta");
    expect(await screen.findByRole("link", { name: "Beta Org" })).toBeInTheDocument();

    const listCallsBeforePinResult = listCalls(fetchMock).length;
    releasePin(jsonResponse({ pinned: true }));

    await waitFor(() => {
      const calls = listCalls(fetchMock).slice(listCallsBeforePinResult);
      expect(calls.length).toBeGreaterThan(0);
      expect(calls.every(([input]) => String(input).includes("search=Beta"))).toBe(true);
    });
    expect(screen.getByRole("link", { name: "Beta Org" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Acme" })).not.toBeInTheDocument();
  });

  it("keeps each organization pin disabled until its own request finishes", async () => {
    const user = userEvent.setup();
    const releases = new Map<string, (response: Response) => void>();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/pin") && init?.method === "PUT") {
        const organizationID = url.split("/").at(-2) ?? "";
        return new Promise<Response>((resolve) => {
          releases.set(organizationID, resolve);
        });
      }
      return jsonResponse({
        items: [organization("org-1", "Acme"), organization("org-2", "Beta")],
        pinned: [],
        total: 2,
        match_total: 2,
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    const [acmePin, betaPin] = await screen.findAllByRole("button", { name: "Pin organization" });
    await user.click(acmePin);
    await user.click(betaPin);

    await waitFor(() => {
      expect(releases.has("org-1")).toBe(true);
      expect(releases.has("org-2")).toBe(true);
    });
    expect(acmePin).toBeDisabled();
    expect(betaPin).toBeDisabled();

    releases.get("org-2")?.(jsonResponse({ pinned: true }));

    await waitFor(() => {
      const pins = screen.getAllByRole("button", { name: "Pin organization" });
      expect(pins[0]).toBeDisabled();
      expect(pins[1]).toBeEnabled();
    });
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === "PUT")).toHaveLength(2);

    releases.get("org-1")?.(jsonResponse({ pinned: true }));
    await waitFor(() => {
      screen.getAllByRole("button", { name: "Pin organization" }).forEach((button) => {
        expect(button).toBeEnabled();
      });
    });
  });

  it("asks to retry when a saved pin cannot refresh the list", async () => {
    const user = userEvent.setup();
    let failNextList = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/pin") && init?.method === "PUT") {
        failNextList = true;
        return jsonResponse({ pinned: true });
      }
      if (failNextList) {
        failNextList = false;
        return jsonResponse({}, 500);
      }
      return jsonResponse({
        items: [organization("org-1", "Acme")],
        pinned: [],
        total: 1,
        match_total: 1,
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Pin organization" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not refresh the organization list.");
    expect(screen.getByRole("button", { name: "Pin organization" })).toBeEnabled();

    fetchMock.mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/pin") && init?.method === "PUT") {
        return jsonResponse({ pinned: true });
      }
      return jsonResponse({
        items: [],
        pinned: [organization("org-1", "Acme")],
        total: 0,
        match_total: 1,
      });
    });
    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByRole("heading", { name: "Pinned" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Unpin organization" })).toBeInTheDocument();
  });

  it("shows remaining hosted credit and sorts by that amount", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (_input: RequestInfo | URL) =>
      jsonResponse({
        items: [
          {
            id: "org-1",
            name: "Acme",
            canvas_count: 2,
            task_count: 5,
            done_task_count: 6,
            member_count: 3,
            workspace_count: 7,
            repository_count: 8,
            remaining_credit_cents: 1250,
            created_at: "2024-01-15T12:00:00Z",
          },
        ],
        pinned: [],
        total: 1,
        match_total: 1,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("button", { name: "Credits" })).toBeInTheDocument();
    expect(screen.getByText("$12.50")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Credits" }));

    await waitFor(() => {
      const urls = fetchMock.mock.calls.map((call) => String(call[0]));
      expect(
        urls.some((url) => url.includes("sort_by=remaining_credit_cents") && url.includes("sort_direction=desc")),
      ).toBe(true);
    });
  });

  it("sorts plans in both directions and retains the sort across search and pagination", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (_input: RequestInfo | URL) =>
      jsonResponse({ items: [organization("org-1", "Acme")], pinned: [], total: 101, match_total: 101 }),
    );
    vi.stubGlobal("fetch", fetchMock);
    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Next" }));
    expect(within(screen.getByRole("row", { name: /Acme/ })).getByText("—")).toBeInTheDocument();
    const lastQuery = () => new URL(String(fetchMock.mock.calls.at(-1)?.[0]), "http://localhost").searchParams;
    expect(lastQuery().get("offset")).toBe("50");

    await user.click(screen.getByRole("button", { name: "Plan" }));
    await waitFor(() => {
      expect(lastQuery().get("sort_by")).toBe("plan_name");
      expect(lastQuery().get("sort_direction")).toBe("asc");
      expect(lastQuery().get("offset")).toBe("0");
    });
    expect(screen.getByRole("columnheader", { name: "Plan" })).toHaveAttribute("aria-sort", "ascending");

    await user.click(screen.getByRole("button", { name: "Plan" }));
    await waitFor(() => expect(lastQuery().get("sort_direction")).toBe("desc"));
    expect(screen.getByRole("columnheader", { name: "Plan" })).toHaveAttribute("aria-sort", "descending");
    await user.type(screen.getByPlaceholderText("Search by name or ID..."), "Acme");
    await waitFor(() => expect(lastQuery().get("search")).toBe("Acme"));
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(lastQuery().get("offset")).toBe("50");
    expect(lastQuery().get("search")).toBe("Acme");
    expect(lastQuery().get("sort_by")).toBe("plan_name");
    expect(lastQuery().get("sort_direction")).toBe("desc");
  });

  it("sorts by workspace and repository counts", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (_input: RequestInfo | URL) =>
      jsonResponse({
        items: [
          {
            id: "org-1",
            name: "Acme",
            canvas_count: 2,
            task_count: 5,
            done_task_count: 6,
            member_count: 3,
            workspace_count: 7,
            repository_count: 8,
            created_at: "2024-01-15T12:00:00Z",
          },
        ],
        total: 1,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    const headers = await screen.findAllByRole("columnheader");
    expect(headers.map((header) => header.textContent)).toEqual([
      "Name",
      "Plan",
      "Automations",
      "Tasks",
      "Done Tasks",
      "Members",
      "Workspaces",
      "Repositories",
      "Credits",
      "Created",
    ]);

    await user.click(screen.getByRole("button", { name: "Workspaces" }));

    await waitFor(() => {
      const urls = fetchMock.mock.calls.map((call) => String(call[0]));
      expect(urls.some((url) => url.includes("sort_by=workspace_count") && url.includes("sort_direction=desc"))).toBe(
        true,
      );
    });

    await user.click(screen.getByRole("button", { name: "Repositories" }));

    await waitFor(() => {
      const urls = fetchMock.mock.calls.map((call) => String(call[0]));
      expect(urls.some((url) => url.includes("sort_by=repository_count") && url.includes("sort_direction=desc"))).toBe(
        true,
      );
    });
  });
});

function listCalls(fetchMock: ReturnType<typeof vi.fn>) {
  return fetchMock.mock.calls.filter(([input]) => String(input).includes("/admin/api/organizations?"));
}

function organization(id: string, name: string) {
  return {
    id,
    name,
    canvas_count: 2,
    task_count: 5,
    done_task_count: 6,
    member_count: 3,
    workspace_count: 7,
    repository_count: 8,
    created_at: "2024-01-15T12:00:00Z",
  };
}

function OrganizationPath() {
  const { pathname } = useLocation();
  return <div>{pathname}</div>;
}
