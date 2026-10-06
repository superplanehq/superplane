import { render, screen, waitFor } from "@testing-library/react";
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
    expect(screen.queryByRole("button", { name: "Canvases" })).not.toBeInTheDocument();
    expect(screen.queryByRole("columnheader", { name: "Description" })).not.toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("6")).toBeInTheDocument();
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
          items: [organization("org-1", "Acme")],
          pinned: [organization("pinned-org", "Pinned Customer")],
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
    created_at: "2024-01-15T12:00:00Z",
  };
}

function OrganizationPath() {
  const { pathname } = useLocation();
  return <div>{pathname}</div>;
}
