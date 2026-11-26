export const PAGE_OBSERVABILITY_ATTRIBUTE = "superplane.page";

export type PageObservabilityContext = {
  pageKey: string;
  attributes: Record<string, string>;
};

export function resolvePageObservability(pathname: string): PageObservabilityContext | null {
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) {
    return { pageKey: "rootOrganizationRedirect", attributes: {} };
  }

  const [first, second] = segments;
  const topLevelPageKey = new Map<string, string>([
    ["login", "login"],
    ["onboarding", "organizationOnboarding"],
    ["setup", "ownerSetup"],
  ]).get(first);

  if (topLevelPageKey) {
    return { pageKey: topLevelPageKey, attributes: {} };
  }

  if (first === "github" && second === "approved") {
    return { pageKey: "githubInstallApproved", attributes: {} };
  }

  if (first === "invite" && second) {
    return { pageKey: "inviteAccept", attributes: { invite_token: second } };
  }

  if (first === "invitations" && second) {
    return { pageKey: "invitationAccept", attributes: {} };
  }

  if (first === "admin") {
    return resolveAdminPageObservability(segments.slice(1));
  }

  if (segments.length === 1) {
    return { pageKey: "organizationHomePage", attributes: { organization_id: first } };
  }

  switch (second) {
    case "apps":
    case "canvases":
      return resolveCanvasPageObservability(first, segments.slice(1));
    case "settings":
      return resolveSettingsPageObservability(first, segments.slice(2));
    case "organization":
      return resolveOrganizationSettingsPageObservability(first, segments.slice(2));
    default:
      return null;
  }
}

function resolveAdminPageObservability(segments: string[]): PageObservabilityContext {
  const [section, organizationId] = segments;

  if (section === "accounts") {
    return { pageKey: "adminAccounts", attributes: {} };
  }

  if (section === "settings") {
    return { pageKey: "adminSettings", attributes: {} };
  }

  if (section === "price-books") {
    return { pageKey: "adminPriceBooks", attributes: {} };
  }

  if (section === "runner-tasks") {
    return { pageKey: "adminRunnerTasks", attributes: {} };
  }

  if (section === "organizations" && organizationId) {
    return { pageKey: "adminOrganizationDetail", attributes: { organization_id: organizationId } };
  }

  return { pageKey: "adminOrganizations", attributes: {} };
}

function resolveCanvasPageObservability(organizationId: string, segments: string[]): PageObservabilityContext | null {
  const [section, canvasId, canvasSection] = segments;
  const organizationAttributes = { organization_id: organizationId };

  if (!canvasId) {
    return null;
  }

  if (section === "apps" && canvasId === "new") {
    return { pageKey: "newApp", attributes: organizationAttributes };
  }

  if (canvasSection === "settings") {
    return {
      pageKey: "canvasSettings",
      attributes: { ...organizationAttributes, canvas_id: canvasId },
    };
  }

  return {
    pageKey: "canvas",
    attributes: { ...organizationAttributes, canvas_id: canvasId },
  };
}

const settingsPageKeys = new Map<string, string>([
  ["general", "settingsGeneral"],
  ["members", "settingsMembers"],
  ["create-group", "settingsCreateGroup"],
  ["roles", "settingsRoles"],
  ["profile", "settingsProfile"],
  ["notifications", "settingsNotifications"],
  ["billing", "settingsUsage"],
]);

function resolveSettingsPageObservability(organizationId: string, segments: string[]): PageObservabilityContext {
  const organizationAttributes = { organization_id: organizationId };

  if (segments.length === 0) {
    return { pageKey: "settingsGeneral", attributes: organizationAttributes };
  }

  const [section, ...rest] = segments;
  const pageKey = settingsPageKeys.get(section);

  if (pageKey) {
    return { pageKey, attributes: organizationAttributes };
  }

  switch (section) {
    case "groups":
      if (rest[0] && rest[1] === "members") {
        return {
          pageKey: "settingsGroupMembers",
          attributes: { ...organizationAttributes, group_name: rest[0] },
        };
      }
      return { pageKey: "settingsGroups", attributes: organizationAttributes };
    case "create-role":
      return {
        pageKey: "settingsCreateRole",
        attributes: rest[0] ? { ...organizationAttributes, role_name: rest[0] } : organizationAttributes,
      };
    case "integrations":
      if (!rest[0]) {
        return { pageKey: "settingsIntegrations", attributes: organizationAttributes };
      }
      if (rest[1] === "setup") {
        return {
          pageKey: "settingsIntegrationSetup",
          attributes: { ...organizationAttributes, integration_name: rest[0] },
        };
      }
      return {
        pageKey: "settingsIntegrationDetail",
        attributes: { ...organizationAttributes, integration_id: rest[0] },
      };
    case "secrets":
      if (rest[0]) {
        return {
          pageKey: "settingsSecretDetail",
          attributes: { ...organizationAttributes, secret_id: rest[0] },
        };
      }
      return { pageKey: "settingsSecrets", attributes: organizationAttributes };
    case "api-keys":
      if (rest[0]) {
        return {
          pageKey: "settingsAPIKeyDetail",
          attributes: { ...organizationAttributes, api_key_id: rest[0] },
        };
      }
      return { pageKey: "settingsAPIKeys", attributes: organizationAttributes };
    default:
      return { pageKey: "settingsUnknown", attributes: organizationAttributes };
  }
}

function resolveOrganizationSettingsPageObservability(
  organizationId: string,
  segments: string[],
): PageObservabilityContext {
  const organizationAttributes = { organization_id: organizationId };
  const [section, ...rest] = segments;

  if (!section || section === "general") {
    return { pageKey: "organizationSettingsGeneral", attributes: organizationAttributes };
  }

  if (section === "workspaces") {
    return { pageKey: "organizationSettingsWorkspaces", attributes: organizationAttributes };
  }

  if (section === "workspace-usage" || section === "llm-spend" || section === "spending") {
    return { pageKey: "organizationSettingsWorkspaceUsage", attributes: organizationAttributes };
  }

  if (section === "billing") {
    return { pageKey: "organizationSettingsBilling", attributes: organizationAttributes };
  }

  if (section === "usage") {
    return { pageKey: "organizationSettingsUsage", attributes: organizationAttributes };
  }

  if (section === "integrations") {
    if (!rest[0]) {
      return { pageKey: "organizationSettingsIntegrations", attributes: organizationAttributes };
    }
    if (rest[1] === "setup") {
      return {
        pageKey: "organizationSettingsIntegrationSetup",
        attributes: { ...organizationAttributes, integration_name: rest[0] },
      };
    }
    return {
      pageKey: "organizationSettingsIntegrationDetail",
      attributes: { ...organizationAttributes, integration_id: rest[0] },
    };
  }

  return { pageKey: "organizationSettingsUnknown", attributes: { ...organizationAttributes, section } };
}
