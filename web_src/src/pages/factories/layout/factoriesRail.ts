export const factoriesRailControlClassName =
  "flex size-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none";

export function isBoardPath(pathname: string): boolean {
  return pathname.includes("/lines");
}

export function isVelocityPath(pathname: string): boolean {
  return pathname.includes("/velocity");
}

export function isSettingsPath(pathname: string): boolean {
  return pathname.includes("/settings");
}

/** One or two letters from a display name, for the icon rail. */
export function initialsForName(name: string): string {
  const parts = name
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0]?.toUpperCase() ?? "");
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0];
  }
  return `${parts[0]}${parts[parts.length - 1]}`;
}
