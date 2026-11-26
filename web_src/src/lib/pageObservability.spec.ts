import { describe, expect, it } from "bun:test";
import { resolvePageObservability } from "@/lib/pageObservability";

describe("resolvePageObservability", () => {
  it("maps top-level routes", () => {
    expect(resolvePageObservability("/")).toEqual({ pageKey: "rootOrganizationRedirect", attributes: {} });
    expect(resolvePageObservability("/login")).toEqual({ pageKey: "login", attributes: {} });
    expect(resolvePageObservability("/onboarding")).toEqual({ pageKey: "organizationOnboarding", attributes: {} });
    expect(resolvePageObservability("/setup")).toEqual({ pageKey: "ownerSetup", attributes: {} });
    expect(resolvePageObservability("/github/approved")).toEqual({
      pageKey: "githubInstallApproved",
      attributes: {},
    });
  });

  it("maps invite links", () => {
    expect(resolvePageObservability("/invite/abc123")).toEqual({
      pageKey: "inviteAccept",
      attributes: { invite_token: "abc123" },
    });
    expect(resolvePageObservability("/invitations/private-token")).toEqual({
      pageKey: "invitationAccept",
      attributes: {},
    });
  });

  it("maps admin routes", () => {
    expect(resolvePageObservability("/admin")).toEqual({ pageKey: "adminOrganizations", attributes: {} });
    expect(resolvePageObservability("/admin/accounts")).toEqual({ pageKey: "adminAccounts", attributes: {} });
    expect(resolvePageObservability("/admin/settings")).toEqual({ pageKey: "adminSettings", attributes: {} });
    expect(resolvePageObservability("/admin/price-books")).toEqual({ pageKey: "adminPriceBooks", attributes: {} });
    expect(resolvePageObservability("/admin/runner-tasks")).toEqual({ pageKey: "adminRunnerTasks", attributes: {} });
    expect(resolvePageObservability("/admin/unknown")).toEqual({ pageKey: "adminOrganizations", attributes: {} });
    expect(resolvePageObservability("/admin/organizations")).toEqual({ pageKey: "adminOrganizations", attributes: {} });
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
    expect(resolvePageObservability("/org-1/canvases/canvas-1/settings")).toEqual({
      pageKey: "canvasSettings",
      attributes: { organization_id: "org-1", canvas_id: "canvas-1" },
    });
    expect(resolvePageObservability("/org-1/apps/new/settings")).toEqual({
      pageKey: "newApp",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/canvases/new")).toEqual({
      pageKey: "canvas",
      attributes: { organization_id: "org-1", canvas_id: "new" },
    });
  });

  it("returns null for missing canvas IDs and unrecognized organization routes", () => {
    expect(resolvePageObservability("/org-1/apps")).toBeNull();
    expect(resolvePageObservability("/org-1/canvases")).toBeNull();
    expect(resolvePageObservability("/org-1/unknown")).toBeNull();
  });

  it("ignores empty and trailing segments in admin and canvas routes", () => {
    expect(resolvePageObservability("//admin//organizations//org-1/ignored/trailing/")).toEqual({
      pageKey: "adminOrganizationDetail",
      attributes: { organization_id: "org-1" },
    });
    for (const route of ["apps", "canvases"]) {
      expect(resolvePageObservability(`/org-1/${route}/canvas-1/ignored/trailing/`)).toEqual({
        pageKey: "canvas",
        attributes: { organization_id: "org-1", canvas_id: "canvas-1" },
      });
      expect(resolvePageObservability(`/org-1/${route}/canvas-1/settings/ignored/trailing/`)).toEqual({
        pageKey: "canvasSettings",
        attributes: { organization_id: "org-1", canvas_id: "canvas-1" },
      });
    }
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
    expect(resolvePageObservability("/org-1/settings/notifications")).toEqual({
      pageKey: "settingsNotifications",
      attributes: { organization_id: "org-1" },
    });
    for (const [section, pageKey] of [
      ["members", "settingsMembers"],
      ["create-group", "settingsCreateGroup"],
      ["roles", "settingsRoles"],
      ["profile", "settingsProfile"],
      ["billing", "settingsUsage"],
      ["general/ignored/trailing", "settingsGeneral"],
      ["unknown", "settingsUnknown"],
      ["toString", "settingsUnknown"],
      ["groups", "settingsGroups"],
      ["groups/admins", "settingsGroups"],
      ["groups/admins/unknown", "settingsGroups"],
      ["create-role", "settingsCreateRole"],
      ["integrations", "settingsIntegrations"],
      ["secrets", "settingsSecrets"],
      ["api-keys", "settingsAPIKeys"],
    ]) {
      expect(resolvePageObservability(`/org-1/settings/${section}`)).toEqual({
        pageKey,
        attributes: { organization_id: "org-1" },
      });
    }
    expect(resolvePageObservability("/org-1/settings")).toEqual({
      pageKey: "settingsGeneral",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/settings/create-role/operator")).toEqual({
      pageKey: "settingsCreateRole",
      attributes: { organization_id: "org-1", role_name: "operator" },
    });
    expect(resolvePageObservability("/org-1/settings/integrations/int-9")).toEqual({
      pageKey: "settingsIntegrationDetail",
      attributes: { organization_id: "org-1", integration_id: "int-9" },
    });
    expect(resolvePageObservability("/org-1/settings/secrets/secret-1")).toEqual({
      pageKey: "settingsSecretDetail",
      attributes: { organization_id: "org-1", secret_id: "secret-1" },
    });
    expect(resolvePageObservability("/org-1/settings/api-keys/key-1")).toEqual({
      pageKey: "settingsAPIKeyDetail",
      attributes: { organization_id: "org-1", api_key_id: "key-1" },
    });
  });

  it("maps factories organization settings routes", () => {
    expect(resolvePageObservability("/org-1/organization/general")).toEqual({
      pageKey: "organizationSettingsGeneral",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/organization/workspace-usage")).toEqual({
      pageKey: "organizationSettingsWorkspaceUsage",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/organization/llm-spend")).toEqual({
      pageKey: "organizationSettingsWorkspaceUsage",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/organization/spending")).toEqual({
      pageKey: "organizationSettingsWorkspaceUsage",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/organization/billing")).toEqual({
      pageKey: "organizationSettingsBilling",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/organization/usage")).toEqual({
      pageKey: "organizationSettingsUsage",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/organization/integrations")).toEqual({
      pageKey: "organizationSettingsIntegrations",
      attributes: { organization_id: "org-1" },
    });
    expect(resolvePageObservability("/org-1/organization/integrations/slack/setup")).toEqual({
      pageKey: "organizationSettingsIntegrationSetup",
      attributes: { organization_id: "org-1", integration_name: "slack" },
    });
    expect(resolvePageObservability("/org-1/organization/integrations/int-9")).toEqual({
      pageKey: "organizationSettingsIntegrationDetail",
      attributes: { organization_id: "org-1", integration_id: "int-9" },
    });
  });
});
