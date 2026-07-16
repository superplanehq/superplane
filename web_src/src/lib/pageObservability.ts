export const PAGE_OBSERVABILITY_ATTRIBUTE = "superplane.page";

export type PageObservabilityContext = {
  pageKey: string;
  attributes: Record<string, string>;
};

export function resolvePageObservability(pathname: string): PageObservabilityContext | null {
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) {
    return { pageKey: "organizationSelect", attributes: {} };
  }

  const [first, second, third, fourth] = segments;

  if (first === "login") {
    return { pageKey: "login", attributes: {} };
  }

  if (first === "create") {
    return { pageKey: "organizationCreate", attributes: {} };
  }

  if (first === "setup") {
    return { pageKey: "ownerSetup", attributes: {} };
  }

  if (first === "install") {
    return { pageKey: "install", attributes: {} };
  }

  if (first === "invite" && second) {
    return { pageKey: "inviteAccept", attributes: { invite_token: second } };
  }

  if (first === "admin") {
    if (second === "accounts") {
      return { pageKey: "adminAccounts", attributes: {} };
    }

    if (second === "settings") {
      return { pageKey: "adminSettings", attributes: {} };
    }

    if (second === "runner-tasks") {
      return { pageKey: "adminRunnerTasks", attributes: {} };
    }

    if (second === "organizations" && third) {
      return { pageKey: "adminOrganizationDetail", attributes: { organization_id: third } };
    }

    return { pageKey: "adminOrganizations", attributes: {} };
  }

  const organizationId = first;
  const organizationAttributes = { organization_id: organizationId };

  if (segments.length === 1) {
    return { pageKey: "organizationHomePage", attributes: organizationAttributes };
  }

  if (second === "apps") {
    if (third === "new") {
      return { pageKey: "newApp", attributes: organizationAttributes };
    }

    if (third && fourth === "settings") {
      return {
        pageKey: "canvasSettings",
        attributes: { ...organizationAttributes, canvas_id: third },
      };
    }

    if (third) {
      return {
        pageKey: "canvas",
        attributes: { ...organizationAttributes, canvas_id: third },
      };
    }
  }

  if (second === "canvases" && third) {
    if (fourth === "settings") {
      return {
        pageKey: "canvasSettings",
        attributes: { ...organizationAttributes, canvas_id: third },
      };
    }

    return {
      pageKey: "canvas",
      attributes: { ...organizationAttributes, canvas_id: third },
    };
  }

  if (second === "settings") {
    return resolveSettingsPageObservability(organizationId, segments.slice(2));
  }

  return null;
}

function resolveSettingsPageObservability(organizationId: string, segments: string[]): PageObservabilityContext {
  const organizationAttributes = { organization_id: organizationId };

  if (segments.length === 0) {
    return { pageKey: "settingsGeneral", attributes: organizationAttributes };
  }

  const [section, ...rest] = segments;

  switch (section) {
    case "general":
      return { pageKey: "settingsGeneral", attributes: organizationAttributes };
    case "members":
      return { pageKey: "settingsMembers", attributes: organizationAttributes };
    case "groups":
      if (rest[0] && rest[1] === "members") {
        return {
          pageKey: "settingsGroupMembers",
          attributes: { ...organizationAttributes, group_name: rest[0] },
        };
      }
      return { pageKey: "settingsGroups", attributes: organizationAttributes };
    case "create-group":
      return { pageKey: "settingsCreateGroup", attributes: organizationAttributes };
    case "roles":
      return { pageKey: "settingsRoles", attributes: organizationAttributes };
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
    case "profile":
      return { pageKey: "settingsProfile", attributes: organizationAttributes };
    case "billing":
      return { pageKey: "settingsUsage", attributes: organizationAttributes };
    default:
      return { pageKey: "settingsUnknown", attributes: organizationAttributes };
  }
}
