import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("api-interceptor", () => {
  let originalFetch: typeof globalThis.fetch;
  let locationHref: string;
  let pathname = "/dashboard";
  let search = "?tab=overview";

  beforeEach(() => {
    vi.resetModules();
    originalFetch = globalThis.fetch;
    locationHref = "http://localhost/dashboard?tab=overview";
    pathname = "/dashboard";
    search = "?tab=overview";

    vi.stubGlobal(
      "window",
      Object.assign(globalThis.window, {
        location: {
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
      }),
    );
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.unstubAllGlobals();
  });

  it("redirects unauthorized api requests on non-auth routes", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    await expect(globalThis.fetch("/api/me")).rejects.toThrow("Unauthorized");
    expect(locationHref).toBe("/login?redirect=%2Fdashboard%3Ftab%3Doverview");
  });

  it("does not redirect non-api requests", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    const response = await globalThis.fetch("/assets/logo.svg");
    expect(response.status).toBe(401);
    expect(locationHref).toBe("http://localhost/dashboard?tab=overview");
  });

  it("does not redirect auth routes", async () => {
    pathname = "/login";
    search = "";
    globalThis.fetch = vi.fn().mockResolvedValue(new Response("", { status: 401 }));
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();

    await expect(globalThis.fetch("/api/me")).rejects.toThrow("Unauthorized");
    expect(locationHref).toBe("http://localhost/dashboard?tab=overview");
  });

  it("wraps fetch only once", async () => {
    const baseFetch = vi.fn().mockResolvedValue(new Response("", { status: 200 }));
    globalThis.fetch = baseFetch;
    const { setupApiInterceptor } = await import("@/lib/api-interceptor");

    setupApiInterceptor();
    const wrappedFetch = globalThis.fetch;
    setupApiInterceptor();

    expect(globalThis.fetch).toBe(wrappedFetch);
  });
});
