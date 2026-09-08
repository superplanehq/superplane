import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Login } from "./Login";
import { getSignupUnavailableReason } from "./signupUnavailableReason";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: null, loading: false }),
}));

vi.mock("@/hooks/useReportPageReady", () => ({
  useReportPageReady: () => undefined,
}));

const authConfig = {
  providers: [] as string[],
  passwordLoginEnabled: false,
  signupEnabled: true,
  signupsBlockedByEnvironment: false,
  magicCodeEnabled: true,
};

function mockAuthConfig() {
  return {
    ok: true,
    json: async () => authConfig,
  } as Response;
}

function renderLogin(path = "/login", mode?: "login" | "signup") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Login mode={mode} />
    </MemoryRouter>,
  );
}

describe("getSignupUnavailableReason", () => {
  it("does not use waitlist state when signup is available", () => {
    expect(getSignupUnavailableReason(false, false, true)).toBeNull();
  });

  it("uses waitlist state only when the waitlist config is complete", () => {
    expect(getSignupUnavailableReason(true, false, true)).toBe("waitlist");
  });

  it("uses closed state when waitlist config is incomplete", () => {
    expect(getSignupUnavailableReason(true, false, false)).toBe("closed");
  });

  it("uses closed state when signups are blocked by environment", () => {
    expect(getSignupUnavailableReason(true, true, true)).toBe("closed");
  });
});

describe("Login magic code signup required", () => {
  beforeEach(() => {
    authConfig.magicCodeEnabled = true;
    authConfig.signupEnabled = true;
    vi.restoreAllMocks();
  });

  it("shows the create prompt when a login code request returns signup_required", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/auth/config") {
        return mockAuthConfig();
      }

      if (url === "/auth/magic-code/request") {
        return {
          ok: false,
          status: 403,
          text: async () => JSON.stringify({ error: "signup_required" }),
        } as Response;
      }

      throw new Error(`unexpected fetch ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderLogin();

    await screen.findByRole("button", { name: "Continue with email" });
    await user.type(screen.getByPlaceholderText("you@example.com"), "new@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with email" }));

    expect(await screen.findByText("No account found")).toBeInTheDocument();
    expect(screen.getByText("This email does not have a SuperPlane account.")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Enter 6-digit code")).not.toBeInTheDocument();
    expect(screen.queryByText("signup must be started from the signup page")).not.toBeInTheDocument();
  });

  it("sends the code with signup intent after Create account", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url === "/auth/config") {
        return mockAuthConfig();
      }

      if (url === "/auth/magic-code/request") {
        const body = String(init?.body ?? "");
        if (body.includes("signup=true")) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ message: "sent" }),
            text: async () => JSON.stringify({ message: "sent" }),
          } as Response;
        }

        return {
          ok: false,
          status: 403,
          text: async () => JSON.stringify({ error: "signup_required" }),
        } as Response;
      }

      throw new Error(`unexpected fetch ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderLogin();

    await screen.findByRole("button", { name: "Continue with email" });
    await user.type(screen.getByPlaceholderText("you@example.com"), "new@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with email" }));
    await screen.findByRole("button", { name: "Create account" });
    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("Check your email")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Enter 6-digit code")).toBeInTheDocument();

    const signupRequest = fetchMock.mock.calls.find(([, init]) => String(init?.body ?? "").includes("signup=true"));
    expect(signupRequest).toBeDefined();
  });

  it("opens the create prompt when verify returns signup_required", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/auth/config") {
        return mockAuthConfig();
      }

      if (url === "/auth/magic-code/request") {
        return {
          ok: true,
          status: 200,
          json: async () => ({ message: "sent" }),
          text: async () => JSON.stringify({ message: "sent" }),
        } as Response;
      }

      if (String(url).startsWith("/auth/magic-code/verify")) {
        return {
          ok: false,
          status: 403,
          clone: () => ({
            text: async () => JSON.stringify({ error: "signup_required" }),
          }),
          text: async () => JSON.stringify({ error: "signup_required" }),
        } as Response;
      }

      throw new Error(`unexpected fetch ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderLogin("/signup", "signup");

    await screen.findByRole("button", { name: "Continue with email" });
    await user.type(screen.getByPlaceholderText("you@example.com"), "new@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with email" }));
    await screen.findByPlaceholderText("Enter 6-digit code");
    await user.type(screen.getByPlaceholderText("Enter 6-digit code"), "123456");
    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("No account found")).toBeInTheDocument();
    expect(screen.queryByText("signup must be started from the signup page")).not.toBeInTheDocument();
  });

  it("shows the closed notice when signups are blocked", async () => {
    authConfig.signupEnabled = false;
    authConfig.signupsBlockedByEnvironment = true;
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/auth/config") {
        return mockAuthConfig();
      }

      if (url === "/auth/magic-code/request") {
        return {
          ok: false,
          status: 403,
          text: async () => JSON.stringify({ error: "signup_required" }),
        } as Response;
      }

      throw new Error(`unexpected fetch ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderLogin();

    await screen.findByRole("button", { name: "Continue with email" });
    await user.type(screen.getByPlaceholderText("you@example.com"), "new@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with email" }));

    expect(await screen.findByText("Signups are closed")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to sign in" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create account" })).not.toBeInTheDocument();
    expect(screen.queryByText('{"error":"signup_required"}')).not.toBeInTheDocument();
  });
});
