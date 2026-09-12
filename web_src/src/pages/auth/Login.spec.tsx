import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { savePendingSignupAnalyticsPreference } from "@/lib/signupAnalytics";

import { Login } from "./Login";
import { getSignupUnavailableReason, isSignupDisabledErrorBody } from "./signupUnavailableReason";

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

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID?: string;
};

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

  it("recognizes the structured signup-disabled response", () => {
    expect(isSignupDisabledErrorBody('{"error":"signup_disabled"}')).toBe(true);
    expect(isSignupDisabledErrorBody('{"error":"account_blocked"}')).toBe(false);
    expect(isSignupDisabledErrorBody("signup is currently disabled")).toBe(false);
  });
});

describe("Login automatic magic-code signup", () => {
  beforeEach(() => {
    authConfig.providers = [];
    authConfig.passwordLoginEnabled = false;
    authConfig.magicCodeEnabled = true;
    authConfig.signupEnabled = true;
    authConfig.signupsBlockedByEnvironment = false;
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("continues directly to code entry for an unknown email", async () => {
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
          text: async () => JSON.stringify({ message: "sent" }),
        } as Response;
      }

      throw new Error(`unexpected fetch ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderLogin();

    await screen.findByRole("button", { name: "Continue with email" });
    await user.type(screen.getByPlaceholderText("you@example.com"), "new@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with email" }));

    expect(await screen.findByText("Check your email")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Enter 6-digit code")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.queryByText("No account found")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create account" })).not.toBeInTheDocument();
    expect(screen.queryByText(/By creating an account, you agree to the/)).not.toBeInTheDocument();
  });

  it("clears an abandoned signup preference before provider login", async () => {
    authConfig.providers = ["google"];
    savePendingSignupAnalyticsPreference({ productUpdatesOptIn: true });
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") {
          return mockAuthConfig();
        }
        throw new Error(`unexpected fetch ${url}`);
      }),
    );

    renderLogin();

    const providerLink = await screen.findByRole("link", { name: "Continue with Google" });
    expect(localStorage).toHaveLength(1);
    providerLink.addEventListener("click", (event) => event.preventDefault());

    await userEvent.click(providerLink);

    expect(localStorage.getItem("superplane:pending_signup_analytics_preference")).toBeNull();
  });

  it("requests an email code when signup preference cleanup fails", async () => {
    const user = userEvent.setup();
    vi.spyOn(Storage.prototype, "removeItem").mockImplementation(() => {
      throw new Error("storage unavailable");
    });
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/auth/config") {
        return mockAuthConfig();
      }
      if (url === "/auth/magic-code/request") {
        return {
          ok: true,
          status: 200,
          text: async () => JSON.stringify({ message: "sent" }),
        } as Response;
      }
      throw new Error(`unexpected fetch ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    renderLogin();

    await screen.findByRole("button", { name: "Continue with email" });
    await user.type(screen.getByPlaceholderText("you@example.com"), "new@example.com");
    await user.click(screen.getByRole("button", { name: "Continue with email" }));

    expect(fetchMock).toHaveBeenCalledWith("/auth/magic-code/request", expect.objectContaining({ method: "POST" }));
  });

  it("shows the closed notice when signup becomes disabled during verification", async () => {
    authConfig.signupsBlockedByEnvironment = true;
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
            text: async () => JSON.stringify({ error: "signup_disabled" }),
          }),
          text: async () => JSON.stringify({ error: "signup_disabled" }),
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

    expect(await screen.findByText("Signups are closed")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to sign in" })).toBeInTheDocument();
    expect(screen.queryByText('{"error":"signup_disabled"}')).not.toBeInTheDocument();
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
          text: async () => JSON.stringify({ error: "signup_disabled" }),
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
    expect(screen.queryByText('{"error":"signup_disabled"}')).not.toBeInTheDocument();
  });
});

describe("Login signup-disabled return", () => {
  const waitlistWindow = window as SignupWaitlistWindow;

  beforeEach(() => {
    authConfig.providers = ["google"];
    authConfig.passwordLoginEnabled = false;
    authConfig.magicCodeEnabled = true;
    authConfig.signupEnabled = false;
    authConfig.signupsBlockedByEnvironment = true;
    delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID;
    delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID;
    vi.restoreAllMocks();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") {
          return mockAuthConfig();
        }
        throw new Error(`unexpected fetch ${url}`);
      }),
    );
  });

  afterEach(() => {
    delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID;
    delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID;
  });

  it("shows the closed-signup state after a provider callback", async () => {
    renderLogin("/login?auth_error=signup_disabled&provider=google");

    await screen.findByRole("heading", { name: "Signups are closed" });

    expect(screen.getByRole("link", { name: "Back to sign in" })).toHaveAttribute("href", "/logout");
    expect(screen.queryByRole("link", { name: "Continue with Google" })).not.toBeInTheDocument();
  });

  it("shows the waitlist after a provider callback when it is configured", async () => {
    authConfig.signupsBlockedByEnvironment = false;
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1";

    renderLogin("/login?auth_error=signup_disabled&provider=google");

    await screen.findByRole("heading", { name: "SuperPlane Cloud" });

    expect(screen.getByRole("button", { name: "Notify me" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to sign in" })).toBeInTheDocument();
  });
});

describe("Sign up terms disclosure", () => {
  beforeEach(() => {
    authConfig.signupEnabled = true;
    authConfig.signupsBlockedByEnvironment = false;
    authConfig.passwordLoginEnabled = true;
    authConfig.magicCodeEnabled = true;
    authConfig.providers = [];
    vi.restoreAllMocks();
  });

  it("shows the terms disclosure in direct sign-up mode", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") {
          return mockAuthConfig();
        }
        throw new Error(`unexpected fetch ${url}`);
      }),
    );

    renderLogin("/signup", "signup");

    await screen.findByRole("heading", { name: "Create your account" });

    expect(screen.getByText(/By creating an account, you agree to the/)).toBeInTheDocument();

    const tosLink = screen.getByRole("link", { name: "Terms of Service" });
    expect(tosLink).toHaveAttribute("href", "https://superplane.com/terms/");
    expect(tosLink).toHaveAttribute("target", "_blank");
    expect(tosLink).toHaveAttribute("rel", "noopener noreferrer");

    const privacyLink = screen.getByRole("link", { name: "Privacy Policy" });
    expect(privacyLink).toHaveAttribute("href", "https://superplane.com/privacy/");
    expect(privacyLink).toHaveAttribute("target", "_blank");
    expect(privacyLink).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("hides the terms disclosure on the normal login screen", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") {
          return mockAuthConfig();
        }
        throw new Error(`unexpected fetch ${url}`);
      }),
    );

    renderLogin("/login", "login");

    await screen.findByRole("heading", { name: "Welcome to SuperPlane" });

    expect(screen.queryByText(/By creating an account, you agree to the/)).not.toBeInTheDocument();
  });

  it("hides the terms disclosure when signups are closed", async () => {
    authConfig.signupEnabled = false;
    authConfig.signupsBlockedByEnvironment = true;

    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url === "/auth/config") {
          return mockAuthConfig();
        }
        throw new Error(`unexpected fetch ${url}`);
      }),
    );

    renderLogin("/signup", "signup");

    await screen.findByRole("heading", { name: "Signups are closed" });

    expect(screen.queryByText(/By creating an account, you agree to the/)).not.toBeInTheDocument();
  });

  it("hides the terms disclosure during config loading", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        return new Promise(() => {});
      }),
    );

    renderLogin("/signup", "signup");

    expect(screen.queryByText(/By creating an account, you agree to the/)).not.toBeInTheDocument();
  });
});
