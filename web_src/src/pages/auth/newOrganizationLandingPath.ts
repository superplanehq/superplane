import { newFactoryPath } from "@/pages/factories/lib/factoryPagePaths";

/**
 * A new organization holds no workspace, so the owner starts in workspace
 * setup. The route creates the workspace, then opens the setup steps.
 */
export const newOrganizationLandingPath = (organizationId: string): string => {
  if (!organizationId) {
    return "/";
  }

  return newFactoryPath(organizationId);
};
