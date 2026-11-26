import { getNameInitials } from "@/lib/nameInitials";

export const factoriesRailControlClassName =
  "flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none";

/**
 * The board is the workspace index. Its line pages (edit and setup) also
 * count as the board, so the rail keeps the Board item current there.
 */
export function isBoardPath(pathname: string): boolean {
  const pageSegments = workspacePageSegments(pathname);
  if (!pageSegments) {
    return false;
  }
  return pageSegments.length === 0 || pageSegments[0] === "line";
}

export function isVelocityPath(pathname: string): boolean {
  return pathname.includes("/velocity");
}

/** Path segments after the workspace key, or undefined outside a workspace. */
function workspacePageSegments(pathname: string): string[] | undefined {
  const segments = pathname.split("/").filter(Boolean);
  const workspacesIndex = segments.lastIndexOf("workspaces");
  const keyIndex = workspacesIndex + 1;
  if (workspacesIndex < 0 || keyIndex >= segments.length) {
    return undefined;
  }
  return segments.slice(keyIndex + 1);
}

function workspacePageSegment(pathname: string): string | undefined {
  return workspacePageSegments(pathname)?.[0];
}

export function isAgentPath(pathname: string): boolean {
  return workspacePageSegment(pathname) === "agent";
}

export function isWorkspaceAgentNavVisible({
  permissionsLoading,
  canUpdateFactories,
}: {
  permissionsLoading: boolean;
  canUpdateFactories: boolean;
}): boolean {
  return !permissionsLoading && canUpdateFactories;
}

/** One or two letters from a display name, for the icon rail. */
export function initialsForName(name: string): string {
  return getNameInitials(name) || "?";
}
