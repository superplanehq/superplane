import { beforeEach, describe, expect, it, vi } from "vitest";

const { init, identify, capture, reset } = vi.hoisted(() => ({
  init: vi.fn(),
  identify: vi.fn(),
  capture: vi.fn(),
  reset: vi.fn(),
}));

vi.mock("posthog-js", () => ({
  default: { init, identify, capture, reset },
}));

import { analytics } from "@/lib/analytics";

describe("analytics", () => {
  beforeEach(() => {
    capture.mockClear();
  });

  it("captures org create", () => {
    analytics.orgCreate("org-123");
    expect(capture).toHaveBeenCalledWith("auth:org_create", {
      organization_id: "org-123",
    });
  });

  it("captures canvas create", () => {
    analytics.canvasCreate("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:canvas_create", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures canvas delete", () => {
    analytics.canvasDelete("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:canvas_delete", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures version publish", () => {
    analytics.versionPublish("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:version_publish", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures integration create", () => {
    analytics.integrationCreate("github", "org-123");
    expect(capture).toHaveBeenCalledWith("integration:integration_create", {
      integration_type: "github",
      organization_id: "org-123",
    });
  });

  it("captures member accept", () => {
    analytics.memberAccept("org-123");
    expect(capture).toHaveBeenCalledWith("settings:member_accept", {
      organization_id: "org-123",
    });
  });
});
