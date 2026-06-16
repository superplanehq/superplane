import { beforeEach, describe, expect, it, vi } from "vitest";

const { init } = vi.hoisted(() => ({
  init: vi.fn(),
}));

vi.mock("@dash0/sdk-web", () => ({
  init,
}));

describe("dash0 init", () => {
  beforeEach(() => {
    init.mockClear();
    vi.resetModules();
    delete (window as Window & { SUPERPLANE_DASH0_OTLP_ENDPOINT?: string }).SUPERPLANE_DASH0_OTLP_ENDPOINT;
    delete (window as Window & { SUPERPLANE_DASH0_AUTH_TOKEN?: string }).SUPERPLANE_DASH0_AUTH_TOKEN;
    delete (window as Window & { SUPERPLANE_DASH0_SERVICE_NAME?: string }).SUPERPLANE_DASH0_SERVICE_NAME;
    delete (window as Window & { SUPERPLANE_DASH0_ENVIRONMENT?: string }).SUPERPLANE_DASH0_ENVIRONMENT;
  });

  it("calls init when endpoint and auth token are set", async () => {
    (window as Window & { SUPERPLANE_DASH0_OTLP_ENDPOINT?: string }).SUPERPLANE_DASH0_OTLP_ENDPOINT =
      "https://ingress.us-west-2.aws.dash0.com:4318";
    (window as Window & { SUPERPLANE_DASH0_AUTH_TOKEN?: string }).SUPERPLANE_DASH0_AUTH_TOKEN = "test-token";
    (window as Window & { SUPERPLANE_DASH0_SERVICE_NAME?: string }).SUPERPLANE_DASH0_SERVICE_NAME =
      "superplane-staging";
    (window as Window & { SUPERPLANE_DASH0_ENVIRONMENT?: string }).SUPERPLANE_DASH0_ENVIRONMENT = "staging";

    const dash0 = await import("@/dash0");

    expect(dash0.isDash0Enabled).toBe(true);
    expect(init).toHaveBeenCalledWith(
      expect.objectContaining({
        serviceName: "superplane-staging",
        environment: "staging",
        endpoint: {
          url: "https://ingress.us-west-2.aws.dash0.com:4318",
          authToken: "test-token",
        },
        ignoreUrls: [/\/ws\//, /posthog\.com/],
        pageViewInstrumentation: expect.objectContaining({
          generateMetadata: expect.any(Function),
        }),
      }),
    );
  });

  it("uses default service name when not configured", async () => {
    (window as Window & { SUPERPLANE_DASH0_OTLP_ENDPOINT?: string }).SUPERPLANE_DASH0_OTLP_ENDPOINT =
      "https://ingress.us-west-2.aws.dash0.com:4318";
    (window as Window & { SUPERPLANE_DASH0_AUTH_TOKEN?: string }).SUPERPLANE_DASH0_AUTH_TOKEN = "test-token";

    await import("@/dash0");

    expect(init).toHaveBeenCalledWith(
      expect.objectContaining({
        serviceName: "superplane-web",
      }),
    );
  });

  it("ignores PostHog analytics traffic", async () => {
    (window as Window & { SUPERPLANE_DASH0_OTLP_ENDPOINT?: string }).SUPERPLANE_DASH0_OTLP_ENDPOINT =
      "https://ingress.us-west-2.aws.dash0.com:4318";
    (window as Window & { SUPERPLANE_DASH0_AUTH_TOKEN?: string }).SUPERPLANE_DASH0_AUTH_TOKEN = "test-token";

    await import("@/dash0");

    const ignoreUrls = init.mock.calls[0]?.[0]?.ignoreUrls as RegExp[];
    expect(ignoreUrls.some((pattern) => pattern.test("https://us.i.posthog.com/e/"))).toBe(true);
  });

  it("does not call init when endpoint is missing", async () => {
    (window as Window & { SUPERPLANE_DASH0_AUTH_TOKEN?: string }).SUPERPLANE_DASH0_AUTH_TOKEN = "test-token";

    const dash0 = await import("@/dash0");

    expect(dash0.isDash0Enabled).toBe(false);
    expect(init).not.toHaveBeenCalled();
  });

  it("does not call init when auth token is missing", async () => {
    (window as Window & { SUPERPLANE_DASH0_OTLP_ENDPOINT?: string }).SUPERPLANE_DASH0_OTLP_ENDPOINT =
      "https://ingress.us-west-2.aws.dash0.com:4318";

    const dash0 = await import("@/dash0");

    expect(dash0.isDash0Enabled).toBe(false);
    expect(init).not.toHaveBeenCalled();
  });
});
