import { beforeEach, describe, expect, it, vi } from "bun:test";

const { init, addSignalAttribute, removeSignalAttribute, sendEvent } = vi.hoisted(() => ({
  init: vi.fn(),
  addSignalAttribute: vi.fn(),
  removeSignalAttribute: vi.fn(),
  sendEvent: vi.fn(),
}));

vi.mock("@dash0/sdk-web", () => ({
  init,
  addSignalAttribute,
  removeSignalAttribute,
  sendEvent,
}));

import { initDash0 } from "@/dash0";

type Dash0TestWindow = Window & {
  SUPERPLANE_DASH0_OTLP_ENDPOINT?: string;
  SUPERPLANE_DASH0_AUTH_TOKEN?: string;
  SUPERPLANE_DASH0_SERVICE_NAME?: string;
  SUPERPLANE_DASH0_ENVIRONMENT?: string;
};

function dash0Window(): Dash0TestWindow {
  return window as Dash0TestWindow;
}

function setDash0Window(values: { endpoint?: string; authToken?: string; serviceName?: string; environment?: string }) {
  const win = dash0Window();
  win.SUPERPLANE_DASH0_OTLP_ENDPOINT = values.endpoint;
  win.SUPERPLANE_DASH0_AUTH_TOKEN = values.authToken;
  win.SUPERPLANE_DASH0_SERVICE_NAME = values.serviceName;
  win.SUPERPLANE_DASH0_ENVIRONMENT = values.environment;
}

describe("dash0 init", () => {
  beforeEach(() => {
    init.mockClear();
    setDash0Window({});
  });

  it("calls init when endpoint and auth token are set", () => {
    setDash0Window({
      endpoint: "https://ingress.us-west-2.aws.dash0.com:4318",
      authToken: "test-token",
      serviceName: "superplane-staging",
      environment: "staging",
    });

    expect(initDash0()).toBe(true);
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

  it("uses default service name when not configured", () => {
    setDash0Window({
      endpoint: "https://ingress.us-west-2.aws.dash0.com:4318",
      authToken: "test-token",
    });

    expect(initDash0()).toBe(true);
    expect(init).toHaveBeenCalledWith(
      expect.objectContaining({
        serviceName: "superplane-web",
      }),
    );
  });

  it("ignores PostHog analytics traffic", () => {
    setDash0Window({
      endpoint: "https://ingress.us-west-2.aws.dash0.com:4318",
      authToken: "test-token",
    });

    expect(initDash0()).toBe(true);
    const ignoreUrls = init.mock.calls[0]?.[0]?.ignoreUrls as RegExp[];
    expect(ignoreUrls.some((pattern) => pattern.test("https://us.i.posthog.com/e/"))).toBe(true);
  });

  it("does not call init when endpoint is missing", () => {
    setDash0Window({ authToken: "test-token" });

    expect(initDash0()).toBe(false);
    expect(init).not.toHaveBeenCalled();
  });

  it("does not call init when auth token is missing", () => {
    setDash0Window({ endpoint: "https://ingress.us-west-2.aws.dash0.com:4318" });

    expect(initDash0()).toBe(false);
    expect(init).not.toHaveBeenCalled();
  });
});
