import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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

  it("captures member accept", () => {
    analytics.memberAccept("org-123");
    expect(capture).toHaveBeenCalledWith("settings:member_accept", {
      organization_id: "org-123",
    });
  });

  describe("integration connect events", () => {
    let dateSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
      dateSpy = vi.spyOn(Date, "now");
    });

    afterEach(() => {
      dateSpy.mockRestore();
    });

    it("captures integration connect start from integrations page", () => {
      analytics.integrationConnectStart("github", "integrations_page", "org-123");
      expect(capture).toHaveBeenCalledWith("integration:connect_start", {
        integration: "github",
        source: "integrations_page",
        organization_id: "org-123",
      });
    });

    it("captures integration connect start from node configuration", () => {
      analytics.integrationConnectStart("slack", "node_configuration", "org-456");
      expect(capture).toHaveBeenCalledWith("integration:connect_start", {
        integration: "slack",
        source: "node_configuration",
        organization_id: "org-456",
      });
    });

    it("captures integration connect submit with duration when start was recorded", () => {
      dateSpy.mockReturnValueOnce(0).mockReturnValueOnce(3000);
      analytics.integrationConnectStart("github", "integrations_page", "org-123");
      analytics.integrationConnectSubmit("github", "integrations_page", "ready", "org-123");
      expect(capture).toHaveBeenLastCalledWith("integration:connect_submit", {
        integration: "github",
        source: "integrations_page",
        status: "ready",
        duration_s: 3,
        organization_id: "org-123",
      });
    });

    it("captures integration connect submit with pending status", () => {
      dateSpy.mockReturnValueOnce(0).mockReturnValueOnce(5000);
      analytics.integrationConnectStart("github", "node_configuration", "org-123");
      analytics.integrationConnectSubmit("github", "node_configuration", "pending", "org-123");
      expect(capture).toHaveBeenLastCalledWith("integration:connect_submit", {
        integration: "github",
        source: "node_configuration",
        status: "pending",
        duration_s: 5,
        organization_id: "org-123",
      });
    });

    it("captures integration connect submit with error status", () => {
      dateSpy.mockReturnValueOnce(0).mockReturnValueOnce(2000);
      analytics.integrationConnectStart("slack", "integrations_page", "org-456");
      analytics.integrationConnectSubmit("slack", "integrations_page", "error", "org-456");
      expect(capture).toHaveBeenLastCalledWith("integration:connect_submit", {
        integration: "slack",
        source: "integrations_page",
        status: "error",
        duration_s: 2,
        organization_id: "org-456",
      });
    });

    it("captures integration connect submit without duration when no start was recorded", () => {
      analytics.integrationConnectSubmit("github", "integrations_page", "ready", "org-123");
      expect(capture).toHaveBeenCalledWith("integration:connect_submit", {
        integration: "github",
        source: "integrations_page",
        status: "ready",
        duration_s: undefined,
        organization_id: "org-123",
      });
    });

    it("clears the start time after submit so a second submit has no duration", () => {
      dateSpy.mockReturnValueOnce(0).mockReturnValueOnce(1000).mockReturnValueOnce(2000);
      analytics.integrationConnectStart("github", "integrations_page", "org-123");
      analytics.integrationConnectSubmit("github", "integrations_page", "ready", "org-123");
      capture.mockClear();
      analytics.integrationConnectSubmit("github", "integrations_page", "ready", "org-123");
      expect(capture).toHaveBeenCalledWith("integration:connect_submit", {
        integration: "github",
        source: "integrations_page",
        status: "ready",
        duration_s: undefined,
        organization_id: "org-123",
      });
    });
  });

  describe("integration configure and delete events", () => {
    it("captures integration configure open from integrations page with ready status", () => {
      analytics.integrationConfigureOpen("github", "integrations_page", "ready", "org-123");
      expect(capture).toHaveBeenCalledWith("integration:configure_open", {
        integration: "github",
        source: "integrations_page",
        previous_status: "ready",
        organization_id: "org-123",
      });
    });

    it("captures integration configure open from node configuration with error status", () => {
      analytics.integrationConfigureOpen("slack", "node_configuration", "error", "org-456");
      expect(capture).toHaveBeenCalledWith("integration:configure_open", {
        integration: "slack",
        source: "node_configuration",
        previous_status: "error",
        organization_id: "org-456",
      });
    });

    it("captures integration configure open with pending status", () => {
      analytics.integrationConfigureOpen("github", "integrations_page", "pending", "org-123");
      expect(capture).toHaveBeenCalledWith("integration:configure_open", {
        integration: "github",
        source: "integrations_page",
        previous_status: "pending",
        organization_id: "org-123",
      });
    });

    it("captures integration delete", () => {
      analytics.integrationDelete("github", "org-123");
      expect(capture).toHaveBeenCalledWith("integration:integration_delete", {
        integration: "github",
        organization_id: "org-123",
      });
    });
  });

  describe("run item and error events", () => {
    it("captures canvas run item open", () => {
      analytics.canvasRunItemOpen("digitalocean.detachKnowledgeBase", "success", "org-123");
      expect(capture).toHaveBeenCalledWith("canvas:run_item_open", {
        node_ref: "digitalocean.detachKnowledgeBase",
        execution_status: "success",
        organization_id: "org-123",
      });
    });

    it("captures canvas run item tab view - details", () => {
      analytics.canvasRunItemTabView("details", "org-123");
      expect(capture).toHaveBeenCalledWith("canvas:run_item_tab_view", { tab: "details", organization_id: "org-123" });
    });

    it("captures canvas run item tab view - payload", () => {
      analytics.canvasRunItemTabView("payload", "org-123");
      expect(capture).toHaveBeenCalledWith("canvas:run_item_tab_view", { tab: "payload", organization_id: "org-123" });
    });

    it("captures canvas run item tab view - config", () => {
      analytics.canvasRunItemTabView("config", "org-123");
      expect(capture).toHaveBeenCalledWith("canvas:run_item_tab_view", { tab: "config", organization_id: "org-123" });
    });

    it("captures canvas component error", () => {
      analytics.canvasComponentError("http.request", "timeout after 30s", "org-123");
      expect(capture).toHaveBeenCalledWith("canvas:component_error", {
        node_ref: "http.request",
        error_message: "timeout after 30s",
        organization_id: "org-123",
      });
    });

    it("captures canvas log view", () => {
      analytics.canvasLogView("org-123");
      expect(capture).toHaveBeenCalledWith("canvas:log_view", { organization_id: "org-123" });
    });
  });
});
