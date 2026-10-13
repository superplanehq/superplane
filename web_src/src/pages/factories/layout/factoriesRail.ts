import { getNameInitials } from "@/lib/nameInitials";

export const factoriesRailControlClassName =
  "flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none";

export function isBoardPath(pathname: string): boolean {
  return pathname.includes("/lines");
}

export function isVelocityPath(pathname: string): boolean {
  return pathname.includes("/velocity");
}

function workspacePageSegment(pathname: string): string | undefined {
  const segments = pathname.split("/");
  const workspacesIndex = segments.lastIndexOf("workspaces");
  if (workspacesIndex < 0) {
    return undefined;
  }
  return segments[workspacesIndex + 2];
}

export function isAgentPath(pathname: string): boolean {
  return workspacePageSegment(pathname) === "agent";
}

export function isWorkspaceAgentNavVisible({
  permissionsLoading,
  canUpdateFactories,
  workspaceAgentEnabled,
}: {
  permissionsLoading: boolean;
  canUpdateFactories: boolean;
  workspaceAgentEnabled: boolean;
}): boolean {
  if (permissionsLoading || !canUpdateFactories) {
    return false;
  }
  return workspaceAgentEnabled;
}

/** One or two letters from a display name, for the icon rail. */
export function initialsForName(name: string): string {
  return getNameInitials(name) || "?";
}
