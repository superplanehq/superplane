import { describe, expect, it } from "vitest";
import { getRouteContext } from "./route";

describe("getRouteContext", () => {
  it("parses app routes as canvas context", () => {
    expect(getRouteContext("/org-1/apps/canvas-1")).toEqual({
      organizationId: "org-1",
      canvasId: "canvas-1",
    });
  });

  it("does not treat apps/new as a canvas route", () => {
    expect(getRouteContext("/org-1/apps/new")).toEqual({
      organizationId: "org-1",
      canvasId: null,
    });
  });

  it("does not treat apps/new/settings as a canvas route", () => {
    expect(getRouteContext("/org-1/apps/new/settings")).toEqual({
      organizationId: "org-1",
      canvasId: null,
    });
  });

  it("still parses legacy canvas routes", () => {
    expect(getRouteContext("/org-1/canvases/canvas-1")).toEqual({
      organizationId: "org-1",
      canvasId: "canvas-1",
    });
  });

  it("does not treat public top-level routes as organization context", () => {
    const publicPaths = [
      "/",
      "/admin",
      "/create",
      "/invite/token-1",
      "/install",
      "/login",
      "/setup",
      "/signup",
      "/welcome",
    ];
    for (const path of publicPaths) {
      expect(getRouteContext(path)).toEqual({ organizationId: null, canvasId: null });
    }
  });
});
