import { describe, expect, it } from "vitest";
import { getRouteContext } from "./route";

describe("getRouteContext", () => {
  it("parses app routes as canvas context", () => {
    expect(getRouteContext("/org-1/apps/canvas-1")).toEqual({
      organizationId: "org-1",
      canvasId: "canvas-1",
      isTemplateRoute: false,
    });
  });

  it("does not treat apps/new as a canvas route", () => {
    expect(getRouteContext("/org-1/apps/new")).toEqual({
      organizationId: "org-1",
      canvasId: null,
      isTemplateRoute: false,
    });
  });

  it("does not treat apps/new/settings as a canvas route", () => {
    expect(getRouteContext("/org-1/apps/new/settings")).toEqual({
      organizationId: "org-1",
      canvasId: null,
      isTemplateRoute: false,
    });
  });

  it("still parses legacy canvas routes", () => {
    expect(getRouteContext("/org-1/canvases/canvas-1")).toEqual({
      organizationId: "org-1",
      canvasId: "canvas-1",
      isTemplateRoute: false,
    });
  });

  it("parses template routes", () => {
    expect(getRouteContext("/org-1/templates/template-1")).toEqual({
      organizationId: "org-1",
      canvasId: "template-1",
      isTemplateRoute: true,
    });
  });
});
