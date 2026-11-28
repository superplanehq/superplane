import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { Login } from "./Login";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: null, loading: false }),
}));

vi.mock("@/hooks/useReportPageReady", () => ({
  useReportPageReady: () => undefined,
}));

const authConfig = {
  providers: [] as string[],
  passwordLoginEnabled: true,
  signupEnabled: true,
  signupsBlockedByEnvironment: false,
  magicCodeEnabled: true,
  ssoEnabled: true,
};

function mockAuthConfig() {
  return {
    ok: true,
    json: async () => authConfig,
  } as Response;
}

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={["/login"]}>
      <Login />
    </MemoryRouter>,
  );
}

function discoverResponse() {
  return {
    ok: true,
    json: async () => ({ ssoAvailable: true, startUrl: "/auth/sso/1/start", required: false }),
  } as Response;
}

describe("Login SSO prompt", () => {
  beforeEach(() => {
    authConfig.magicCodeEnabled = true;
    authConfig.ssoEnabled = true;
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("drops the prompt when the password email changes", async () => {
    authConfig.magicCodeEnabled = false;
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") return mockAuthConfig();
        if (url === "/auth/sso/discover") return discoverResponse();
        throw new Error(`unexpected fetch ${url}`);
      }),
    );
    renderLogin();
    await user.type(await screen.findByPlaceholderText("Email"), "ada@a.example");
    await user.type(screen.getByPlaceholderText("Password"), "secret");
    await user.click(screen.getByRole("button", { name: "Login" }));
    expect(await screen.findByText("Your organization uses SSO.")).toBeInTheDocument();
    await user.clear(screen.getByPlaceholderText("Email"));
    await user.type(screen.getByPlaceholderText("Email"), "ada@b.example");
    expect(screen.queryByText("Your organization uses SSO.")).not.toBeInTheDocument();
  });

  it("discovers SSO again after another method was chosen for a different email", async () => {
    authConfig.magicCodeEnabled = false;
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") return mockAuthConfig();
        if (url === "/auth/sso/discover") return discoverResponse();
        if (url === "/login") return { ok: false, status: 401, text: async () => "" } as Response;
        throw new Error(`unexpected fetch ${url}`);
      }),
    );
    renderLogin();
    await user.type(await screen.findByPlaceholderText("Email"), "ada@a.example");
    await user.type(screen.getByPlaceholderText("Password"), "secret");
    await user.click(screen.getByRole("button", { name: "Login" }));
    expect(await screen.findByText("Your organization uses SSO.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Use another method" }));
    expect(await screen.findByText("Invalid email or password")).toBeInTheDocument();
    await user.clear(screen.getByPlaceholderText("Email"));
    await user.type(screen.getByPlaceholderText("Email"), "ada@b.example");
    await user.click(screen.getByRole("button", { name: "Login" }));
    expect(await screen.findByText("Your organization uses SSO.")).toBeInTheDocument();
  });

  it("drops the prompt when the magic code email changes", async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") return mockAuthConfig();
        if (url === "/auth/sso/discover") return discoverResponse();
        throw new Error(`unexpected fetch ${url}`);
      }),
    );
    renderLogin();
    await user.type(await screen.findByPlaceholderText("you@example.com"), "ada@a.example");
    await user.click(screen.getByRole("button", { name: "Continue with email" }));
    expect(await screen.findByText("Your organization uses SSO.")).toBeInTheDocument();
    await user.clear(screen.getByPlaceholderText("you@example.com"));
    await user.type(screen.getByPlaceholderText("you@example.com"), "ada@b.example");
    expect(screen.queryByText("Your organization uses SSO.")).not.toBeInTheDocument();
  });
});
