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
});
