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

  it("captures canvas view", () => {
    analytics.canvasView("canvas-123", 5, 3, "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:canvas_view", {
      canvas_id: "canvas-123",
      node_count: 5,
      edge_count: 3,
      organization_id: "org-123",
    });
  });

  it("captures canvas create", () => {
    analytics.canvasCreate("canvas-123", "org-123", "ui", undefined, false);
    expect(capture).toHaveBeenCalledWith("canvas:canvas_create", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
      method: "ui",
      template_id: undefined,
      has_description: false,
    });
  });

  it("captures canvas create from template", () => {
    analytics.canvasCreate("canvas-123", "org-123", "template", "template-456", true);
    expect(capture).toHaveBeenCalledWith("canvas:canvas_create", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
      method: "template",
      template_id: "template-456",
      has_description: true,
    });
  });

  it("captures canvas delete", () => {
    analytics.canvasDelete("canvas-123", "org-123", 5);
    expect(capture).toHaveBeenCalledWith("canvas:canvas_delete", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
      node_count: 5,
    });
  });

  it("captures canvas rename", () => {
    analytics.canvasRename("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:canvas_rename", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures yaml export", () => {
    analytics.yamlExport("canvas-123", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:yaml_export", {
      canvas_id: "canvas-123",
      organization_id: "org-123",
    });
  });

  it("captures yaml import", () => {
    analytics.yamlImport();
    expect(capture).toHaveBeenCalledWith("canvas:yaml_import", {});
  });

  it("captures node add action", () => {
    analytics.nodeAdd("action", "github", "github.create_issue", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:node_add", {
      node_type: "action",
      integration: "github",
      node_ref: "github.create_issue",
      organization_id: "org-123",
    });
  });

  it("captures node add trigger", () => {
    analytics.nodeAdd("trigger", undefined, "cron.scheduled", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:node_add", {
      node_type: "trigger",
      integration: undefined,
      node_ref: "cron.scheduled",
      organization_id: "org-123",
    });
  });

  it("captures node remove", () => {
    analytics.nodeRemove("action", "slack", "slack.send_message", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:node_remove", {
      node_type: "action",
      integration: "slack",
      node_ref: "slack.send_message",
      organization_id: "org-123",
    });
  });

  it("captures node configure", () => {
    analytics.nodeConfigure("action", "github", 3, "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:node_configure", {
      node_type: "action",
      integration: "github",
      field_count: 3,
      organization_id: "org-123",
    });
  });

  it("captures edge create", () => {
    analytics.edgeCreate("org-123");
    expect(capture).toHaveBeenCalledWith("canvas:edge_create", {
      organization_id: "org-123",
    });
  });

  it("captures edge remove", () => {
    analytics.edgeRemove("org-123");
    expect(capture).toHaveBeenCalledWith("canvas:edge_remove", {
      organization_id: "org-123",
    });
  });

  it("captures auto layout", () => {
    analytics.autoLayout(5, "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:auto_layout", {
      node_count: 5,
      organization_id: "org-123",
    });
  });

  it("captures event emit", () => {
    analytics.eventEmit("trigger", "github", "org-123");
    expect(capture).toHaveBeenCalledWith("canvas:event_emit", {
      node_type: "trigger",
      integration: "github",
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
