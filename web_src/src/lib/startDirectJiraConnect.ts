import type { OrganizationsCreateIntegrationResponse } from "@/api-client";

import { followBrowserAction } from "@/lib/browserAction";
import { INTEGRATION_SETUP_RETURN_PATH_KEY, rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { createWithGeneratedName } from "@/ui/IntegrationCreateDialog/generatedName";

type StartDirectJiraConnectArgs = {
  organizationId: string;
  returnTo?: string;
  existingNames: Set<string>;
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

/** Creates a Jira connection and opens Atlassian authorization in this tab. */
export async function startDirectJiraConnect(args: StartDirectJiraConnectArgs): Promise<boolean> {
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
