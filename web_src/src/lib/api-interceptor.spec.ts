import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";
import { ACCOUNT_BLOCKED_MESSAGE } from "@/lib/account-blocked";
import { client } from "@/api-client/client.gen";

describe("api-interceptor", () => {
  let originalFetch: typeof globalThis.fetch;
  let locationHref: string;
  let pathname = "/dashboard";
  let search = "?tab=overview";

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    locationHref = "http://localhost/dashboard?tab=overview";
    pathname = "/dashboard";
    search = "?tab=overview";

    Object.defineProperty(window, "location", {
      configurable: true,
      enumerable: true,
      value: {
        get pathname() {
          return pathname;
        },
        get search() {
          return search;
        },
        get href() {
          return locationHref;
        },
        set href(value: string) {
          locationHref = value;
        },
      },
    });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("redirects unauthorized api requests on non-auth routes", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    await expect(globalThis.fetch("/api/me")).rejects.toThrow("Unauthorized");
    expect(locationHref).toBe("/login?redirect=%2Fdashboard%3Ftab%3Doverview");
  });

  it("redirects blocked api requests to the login message", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response(ACCOUNT_BLOCKED_MESSAGE, { status: 403 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    await expect(globalThis.fetch("/api/me")).rejects.toThrow(ACCOUNT_BLOCKED_MESSAGE);
    expect(locationHref).toBe("/login?auth_error=account_blocked");
  });

  it("redirects blocked account-session requests", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response(ACCOUNT_BLOCKED_MESSAGE, { status: 403 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    const paths = [
      "/account",
      "/account/limits",
      "/account/onboarding",
      "/account/password",
      "/account/experimental-features",
      "/organizations",
    ];
    for (const path of paths) {
      await expect(globalThis.fetch(path)).rejects.toThrow(ACCOUNT_BLOCKED_MESSAGE);
      expect(locationHref).toBe("/login?auth_error=account_blocked");
    }
  });

  it("leaves unrelated forbidden responses unchanged", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response("Forbidden", { status: 403 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    const response = await globalThis.fetch("/api/me");
    expect(response.status).toBe(403);
    expect(locationHref).toBe("http://localhost/dashboard?tab=overview");
  });

  it("does not redirect non-api requests", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    const response = await globalThis.fetch("/assets/logo.svg");
    expect(response.status).toBe(401);
    expect(locationHref).toBe("http://localhost/dashboard?tab=overview");
  });

  it("does not hard-redirect the account session probe on 401", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    const response = await globalThis.fetch("/account");
    expect(response.status).toBe(401);
    expect(locationHref).toBe("http://localhost/dashboard?tab=overview");
  });

  it("redirects an expired member session on a public line URL", async () => {
    pathname = "/demo/workspaces/newwo/lines/fb0e0e21-8d19-4b3e-ac3f-cfe1cc54f4d7";
    search = "";
    locationHref = "http://localhost" + pathname;
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    await expect(globalThis.fetch("/api/v1/me")).rejects.toThrow("Unauthorized");
    expect(locationHref).toBe(
      "/login?redirect=%2Fdemo%2Fworkspaces%2Fnewwo%2Flines%2Ffb0e0e21-8d19-4b3e-ac3f-cfe1cc54f4d7",
    );
  });

  it("redirects organization list 401 outside a public line", async () => {
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    await expect(globalThis.fetch("/organizations")).rejects.toThrow("Unauthorized");
    expect(locationHref).toBe("/login?redirect=%2Fdashboard%3Ftab%3Doverview");
  });

  it("does not redirect guest probes on a public line URL", async () => {
    pathname = "/demo/workspaces/newwo/lines/fb0e0e21-8d19-4b3e-ac3f-cfe1cc54f4d7";
    search = "";
    locationHref = "http://localhost" + pathname;
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    for (const path of ["/account", "/organizations", "/account/experimental-features"]) {
      const response = await globalThis.fetch(path);
      expect(response.status).toBe(401);
      expect(locationHref).toBe("http://localhost/demo/workspaces/newwo/lines/fb0e0e21-8d19-4b3e-ac3f-cfe1cc54f4d7");
    }
  });

  it("does not redirect auth routes", async () => {
    pathname = "/login";
    search = "";
    globalThis.fetch = mock().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    await expect(globalThis.fetch("/api/me")).rejects.toThrow("Unauthorized");
    expect(locationHref).toBe("http://localhost/dashboard?tab=overview");
  });

  it("wraps fetch only once", async () => {
    const baseFetch = mock().mockResolvedValue(new Response("", { status: 200 }));
    globalThis.fetch = baseFetch;
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();
    const wrappedFetch = globalThis.fetch;
    setupApiInterceptor();

    expect(globalThis.fetch).toBe(wrappedFetch);
  });

  it.each([
    ["Not Found", 404, "Not Found"],
    [JSON.stringify({ message: "Canvas not found", code: "NOT_FOUND" }), 404, "Canvas not found"],
    [JSON.stringify({ message: "Invalid request", code: "INVALID_ARGUMENT" }), 400, "Invalid request"],
  ])("rejects API failure %s with an Error and preserves the status", async (body, status, message) => {
    globalThis.fetch = mock().mockResolvedValue(new Response(body, { status }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");
    setupApiInterceptor();

    const failure = client.get({ url: "http://localhost/api/v1/canvases/missing" });
    await expect(failure).rejects.toBeInstanceOf(Error);
    await expect(failure).rejects.toMatchObject({ message, status });
    if (body.startsWith("{")) {
      await expect(failure).rejects.toMatchObject({ code: JSON.parse(body).code });
    }
    expect(locationHref).toBe("http://localhost/dashboard?tab=overview");
  });

  it("preserves an existing network Error", async () => {
    const error = new Error("Network unavailable");
    globalThis.fetch = mock().mockRejectedValue(error);
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");
    setupApiInterceptor();

    await expect(client.get({ url: "http://localhost/api/v1/canvases/missing" })).rejects.toBe(error);
  });
});
