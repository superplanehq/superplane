import { getNameInitials } from "@/lib/nameInitials";

export const factoriesRailControlClassName =
  "flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none";

export function isBoardPath(pathname: string): boolean {
  return pathname.includes("/lines");
}

export function isVelocityPath(pathname: string): boolean {
  return pathname.includes("/velocity");
}

export function isAgentPath(pathname: string): boolean {
  return pathname.split("/").includes("agent");
}

export function isWorkspaceAgentNavVisible({
  permissionsLoading,
  canUpdateFactories,
  workspaceMcpEnabled,
  workspaceSkillsEnabled,
}: {
  permissionsLoading: boolean;
  canUpdateFactories: boolean;
  workspaceMcpEnabled: boolean;
  workspaceSkillsEnabled: boolean;
}): boolean {
  if (permissionsLoading || !canUpdateFactories) {
    return false;
  }
  return workspaceMcpEnabled || workspaceSkillsEnabled;
}

/** One or two letters from a display name, for the icon rail. */
export function initialsForName(name: string): string {
  return getNameInitials(name) || "?";
}
