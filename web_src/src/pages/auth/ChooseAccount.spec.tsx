import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ChooseAccount } from "./ChooseAccount";

vi.mock("@/hooks/useReportPageReady", () => ({
  useReportPageReady: () => undefined,
}));

function renderChooseAccount(path = "/login/choose-account?token=select:test-token") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <ChooseAccount />
    </MemoryRouter>,
  );
}

describe("ChooseAccount", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("lists matching accounts and opens the selected account", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.startsWith("/auth/choose-account?") && (!init || init.method === undefined || init.method === "GET")) {
        return {
          ok: true,
          json: async () => ({
            accounts: [
              {
                id: "account-a",
                name: "Account A",
                email: "a@example.com",
                avatar_url: "https://avatars.example/a.png",
              },
              {
                id: "account-b",
                name: "Account B",
                email: "b@example.com",
                avatar_url: "",
              },
            ],
          }),
        } as Response;
      }

      if (url === "/auth/choose-account" && init?.method === "POST") {
        expect(String(init.body)).toContain("account_id=account-b");
        expect(String(init.body)).toContain("token=select%3Atest-token");
        return {
          ok: true,
          json: async () => ({ redirectUrl: "/canvases" }),
        } as Response;
      }

      throw new Error(`unexpected fetch ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);
    const location = { href: "" };
    vi.stubGlobal("location", location);

    renderChooseAccount();

    expect(await screen.findByRole("heading", { name: "Choose an account" })).toBeInTheDocument();
    expect(
      screen.getByText("This GitHub identity can sign in to more than one SuperPlane account."),
    ).toBeInTheDocument();
    expect(await screen.findByText("Account A")).toBeInTheDocument();
    expect(screen.getByText("a@example.com")).toBeInTheDocument();
    expect(screen.getByText("Account B")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Account B/ }));

    await waitFor(() => {
      expect(location.href).toBe("/canvases");
    });
  });

  it("asks the person to sign in again when the selection token is missing", async () => {
    renderChooseAccount("/login/choose-account");

    expect(
      await screen.findByText("This page needs a valid account selection link. Sign in again."),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to sign in" })).toHaveAttribute("href", "/login");
  });
});
