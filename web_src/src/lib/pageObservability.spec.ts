import { describe, expect, it } from "vitest";
import { resolvePageObservability } from "@/lib/pageObservability";

describe("resolvePageObservability", () => {
  it("maps top-level routes", () => {
    expect(resolvePageObservability("/")).toEqual({ pageKey: "organizationSelect", attributes: {} });
    expect(resolvePageObservability("/login")).toEqual({ pageKey: "login", attributes: {} });
    expect(resolvePageObservability("/create")).toEqual({ pageKey: "organizationCreate", attributes: {} });
    expect(resolvePageObservability("/setup")).toEqual({ pageKey: "ownerSetup", attributes: {} });
    expect(resolvePageObservability("/install")).toEqual({ pageKey: "install", attributes: {} });
  });

  it("maps invite links", () => {
    expect(resolvePageObservability("/invite/abc123")).toEqual({
      pageKey: "inviteAccept",
      attributes: { invite_token: "abc123" },
    });
  });

  it("maps admin routes", () => {
    expect(resolvePageObservability("/admin")).toEqual({ pageKey: "adminOrganizations", attributes: {} });
    expect(resolvePageObservability("/admin/accounts")).toEqual({ pageKey: "adminAccounts", attributes: {} });
    expect(resolvePageObservability("/admin/organizations/org-1")).toEqual({
      pageKey: "adminOrganizationDetail",
      attributes: { organization_id: "org-1" },
    });
  });

  it("maps organization home and canvas routes", () => {
    expect(resolvePageObservability("/org-1")).toEqual({
      pageKey: "organizationHomePage",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/apps/new")).toEqual({
      pageKey: "newApp",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/apps/canvas-1")).toEqual({
      pageKey: "canvas",
      attributes: { organization_id: "org-1", canvas_id: "canvas-1" },
    });
    expect(resolvePageObservability("/org-1/apps/canvas-1/settings")).toEqual({
      pageKey: "canvasSettings",
      attributes: { organization_id: "org-1", canvas_id: "canvas-1" },
    });
    expect(resolvePageObservability("/org-1/canvases/canvas-1")).toEqual({
      pageKey: "canvas",
      attributes: { organization_id: "org-1", canvas_id: "canvas-1" },
    });
  });

  it("maps settings routes", () => {
    expect(resolvePageObservability("/org-1/settings/general")).toEqual({
      pageKey: "settingsGeneral",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/settings/integrations/slack/setup")).toEqual({
      pageKey: "settingsIntegrationSetup",
      attributes: { organization_id: "org-1", integration_name: "slack" },
    });
    expect(resolvePageObservability("/org-1/settings/groups/admins/members")).toEqual({
      pageKey: "settingsGroupMembers",
      attributes: { organization_id: "org-1", group_name: "admins" },
    });
  });
});
