import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import OrganizationsList from "./OrganizationsList";

const acme = {
  id: "org-1",
  name: "Acme",
  canvas_count: 0,
  task_count: 0,
  done_task_count: 0,
  member_count: 0,
  workspace_count: 0,
  repository_count: 0,
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function listCalls(fetchMock: ReturnType<typeof vi.fn>) {
  return fetchMock.mock.calls.filter(([input]) => String(input).includes("/admin/api/organizations?"));
}

describe("Organization plan filter", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("resets the page for a plan filter and retains it across search, sort, pagination, and pin refresh", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/billing-plans")) {
        return jsonResponse([
          { key: "custom_team", name: "Custom Team" },
          { key: "none", name: "None" },
        ]);
      }
      if (init?.method === "PUT") return jsonResponse({ pinned: true });
      return jsonResponse({ items: [acme], pinned: [], total: 101, match_total: 101 });
    });
    vi.stubGlobal("fetch", fetchMock);
    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    const lastQuery = () => new URL(String(listCalls(fetchMock).at(-1)?.[0]), "http://localhost").searchParams;
    await user.click(await screen.findByRole("button", { name: "Next" }));
    expect(lastQuery().get("offset")).toBe("50");

    await user.click(screen.getByRole("combobox", { name: "Filter by plan" }));
    await user.click(await screen.findByRole("option", { name: "Custom Team" }));
    await waitFor(() => {
      expect(lastQuery().get("plan_key")).toBe("custom_team");
      expect(lastQuery().get("offset")).toBe("0");
    });
    await user.click(screen.getByRole("button", { name: "Name" }));
    await waitFor(() => expect(lastQuery().get("sort_by")).toBe("name"));
    await user.type(screen.getByPlaceholderText("Search by name or ID..."), "Acme");
    await waitFor(() => expect(lastQuery().get("search")).toBe("Acme"));
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(lastQuery().get("offset")).toBe("50");
    expect(lastQuery().get("plan_key")).toBe("custom_team");

    const beforePin = listCalls(fetchMock).length;
    await user.click(screen.getByRole("button", { name: "Pin organization" }));
    await waitFor(() => expect(listCalls(fetchMock).length).toBeGreaterThan(beforePin));
    expect(lastQuery().get("plan_key")).toBe("custom_team");
    expect(lastQuery().get("search")).toBe("Acme");
    expect(lastQuery().get("sort_by")).toBe("name");

    fetchMock.mockImplementation(async () => jsonResponse({ items: [], pinned: [], total: 0, match_total: 0 }));
    await user.click(screen.getByRole("combobox", { name: "Filter by plan" }));
    await user.click(screen.getByRole("option", { name: "None" }));
    expect(await screen.findByText(/No organizations match your filters/)).toBeInTheDocument();
    expect(lastQuery().get("plan_key")).toBe("none");
    expect(screen.getByRole("combobox", { name: "Filter by plan" })).toBeEnabled();

    await user.click(screen.getByRole("combobox", { name: "Filter by plan" }));
    await user.click(screen.getByRole("option", { name: "All plans" }));
    await waitFor(() => {
      expect(lastQuery().has("plan_key")).toBe(false);
      expect(lastQuery().get("offset")).toBe("0");
    });
  });

  it("can retry loading plan options without losing the organization list", async () => {
    const user = userEvent.setup();
    let catalogFailed = false;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).endsWith("/billing-plans")) {
          if (!catalogFailed) {
            catalogFailed = true;
            return jsonResponse({}, 500);
          }
          return jsonResponse([{ key: "trial", name: "Trial" }]);
        }
        return jsonResponse({ items: [acme], total: 1 });
      }),
    );
    render(
      <MemoryRouter>
        <OrganizationsList />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("link", { name: "Acme" })).toBeInTheDocument();
    await user.click(screen.getByRole("combobox", { name: "Filter by plan" }));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByRole("option", { name: "Trial" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.getByRole("link", { name: "Acme" })).toBeInTheDocument();
  });
});
