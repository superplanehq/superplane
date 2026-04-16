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

  it("captures organization created", () => {
    analytics.organizationCreated("org-123");
    expect(capture).toHaveBeenCalledWith("organization created", {
      organization_id: "org-123",
    });
  });

  it("captures canvas created", () => {
    analytics.canvasCreated("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas created", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures canvas deleted", () => {
    analytics.canvasDeleted("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas deleted", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures canvas published", () => {
    analytics.canvasPublished("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas published", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures integration connected", () => {
    analytics.integrationConnected("github", "org-123");
    expect(capture).toHaveBeenCalledWith("integration connected", {
      integration_type: "github",
      organization_id: "org-123",
    });
  });

  it("captures organization joined", () => {
    analytics.organizationJoined("org-123");
    expect(capture).toHaveBeenCalledWith("organization joined", {
      organization_id: "org-123",
    });
  });
});
