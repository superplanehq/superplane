import { render, screen } from "@testing-library/react";
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
});

function OrganizationPath() {
  const { pathname } = useLocation();
  return <div>{pathname}</div>;
}
