import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import AccountsList from "./AccountsList";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "current-account" } }),
}));

afterEach(() => vi.unstubAllGlobals());

describe("AccountsList", () => {
  it("resets pagination when filtering admins and preserves the filter for search, sorting, and paging", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL) =>
      Response.json({
        items: [{ id: "admin", name: "Ada", email: "ada@example.com", installation_admin: true, blocked: false }],
        total: 101,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    render(
      <MemoryRouter>
        <AccountsList />
      </MemoryRouter>,
    );

    const lastQuery = () => new URL(String(fetchMock.mock.calls.at(-1)?.[0]), "http://localhost").searchParams;
    await screen.findByText("Ada");
    expect(lastQuery().has("installation_admin")).toBe(false);
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => expect(lastQuery().get("offset")).toBe("50"));

    fireEvent.click(screen.getByRole("checkbox", { name: "Installation admins only" }));
    await waitFor(() => {
      expect(lastQuery().get("installation_admin")).toBe("true");
      expect(lastQuery().get("offset")).toBe("0");
    });
    fireEvent.change(screen.getByPlaceholderText("Search by name or email..."), { target: { value: "ada" } });
    await waitFor(() => expect(lastQuery().get("search")).toBe("ada"));
    fireEvent.click(screen.getByRole("button", { name: /Name/ }));
    await waitFor(() => expect(lastQuery().get("sort_by")).toBe("name"));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => {
      expect(lastQuery().get("offset")).toBe("50");
      expect(lastQuery().get("installation_admin")).toBe("true");
      expect(lastQuery().get("search")).toBe("ada");
    });

    fireEvent.click(screen.getByRole("checkbox", { name: "Installation admins only" }));
    await waitFor(() => {
      expect(lastQuery().has("installation_admin")).toBe(false);
      expect(lastQuery().get("offset")).toBe("0");
    });
  });

  it("returns to the first page after demoting the last admin on the second page", async () => {
    const admins = Array.from({ length: 51 }, (_, index) => ({
      id: `admin-${index}`,
      name: `Admin ${index}`,
      email: `admin-${index}@example.com`,
      installation_admin: true,
      blocked: false,
    }));
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(String(input), "http://localhost");
      if (init?.method === "POST" && url.pathname === "/admin/api/accounts/admin-50/demote") {
        admins.pop();
        return Response.json({});
      }
      const offset = Number(url.searchParams.get("offset"));
      return Response.json({ items: admins.slice(offset, offset + 50), total: admins.length });
    });
    vi.stubGlobal("fetch", fetchMock);
    render(
      <MemoryRouter>
        <AccountsList />
      </MemoryRouter>,
    );
    await screen.findByText("Admin 0");
    fireEvent.click(screen.getByRole("checkbox", { name: "Installation admins only" }));
    await waitFor(() => {
      const url = new URL(String(fetchMock.mock.calls.at(-1)?.[0]), "http://localhost");
      expect(url.searchParams.get("installation_admin")).toBe("true");
    });
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await screen.findByText("Admin 50");
    fireEvent.click(screen.getByRole("button", { name: "Demote" }));
    fireEvent.click(await screen.findByRole("button", { name: "Remove Admin Access" }));

    await screen.findByText("50 accounts");
    expect(await screen.findByText("Admin 0")).toBeInTheDocument();
    expect(screen.queryByText("Admin 50")).not.toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "Installation admins only" })).toBeChecked();
  });
});
