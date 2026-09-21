import type {
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
  OrganizationsIntegration,
} from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { INTEGRATION_SETUP_RETURN_PATH_KEY, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

type JiraConnectionSummary = Pick<OrganizationsIntegration, "metadata" | "status">;

type StartDirectJiraConnectArgs = {
  organizationId: string;
  returnTo?: string;
  existingNames: Set<string>;
  /** Existing Jira connections in the organization. Used to resume pending OAuth or reuse a ready connection. */
  connected?: JiraConnectionSummary[];
  /** When set, skip reuse/resume and always create a new connection. */
  forceNew?: boolean;
  /** Called when a ready Jira connection already exists instead of opening OAuth. */
  onExistingReady?: (integrationId: string) => void;
  create: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<OrganizationsCreateIntegrationResponse>;
};

function setupReturnConfiguration(returnTo?: string): Record<string, unknown> | undefined {
  if (!returnTo) {
    return undefined;
  }

  return { [INTEGRATION_SETUP_RETURN_PATH_KEY]: returnTo };
}

export function findReadyJiraConnection(connected: JiraConnectionSummary[]): JiraConnectionSummary | undefined {
  return connected.find(
    (item) => item.metadata?.integrationName === "jira" && item.status?.state === "ready" && item.metadata?.id,
  );
}

export function findPendingJiraConnection(connected: JiraConnectionSummary[]): JiraConnectionSummary | undefined {
  return connected.find(
    (item) =>
      item.metadata?.integrationName === "jira" &&
      item.status?.state !== "ready" &&
      Boolean(item.status?.browserAction?.url),
  );
}

async function resumePendingJiraConnect(args: StartDirectJiraConnectArgs): Promise<boolean> {
  if (!args.connected?.length) {
    return false;
  }

  const pending = findPendingJiraConnection(args.connected);
  const action = pending?.status?.browserAction as OrganizationsBrowserAction | undefined;
  if (!action?.url) {
    return false;
  }

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  return followBrowserAction(action);
}

/** Creates a Jira connection and opens Atlassian authorization in this tab. */
export async function startDirectJiraConnect(args: StartDirectJiraConnectArgs): Promise<boolean> {
  if (!args.forceNew && args.connected?.length) {
    const ready = findReadyJiraConnection(args.connected);
    if (ready?.metadata?.id) {
      args.onExistingReady?.(ready.metadata.id);
      return false;
    }

    if (await resumePendingJiraConnect(args)) {
      return true;
    }
  }

  const { result } = await createWithGeneratedName({
    baseName: "jira",
    takenNames: args.existingNames,
    create: (name) => {
      const configuration = setupReturnConfiguration(args.returnTo);
      return args.create({
        integrationName: "jira",
        name,
        ...(configuration ? { configuration } : {}),
      });
    },
  });

  rememberIntegrationSetupReturn(args.organizationId, args.returnTo);
  const action = result.integration?.status?.browserAction;
  if (!action?.url) {
    throw new Error("The Jira authorization page did not open.");
  }
  return followBrowserAction(action);
}
