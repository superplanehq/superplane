import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { Gauge, Kanban, Settings } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link, useLocation } from "react-router";
import { factoryHomePath, factorySettingsWorkspaceGeneralPath, factoryVelocityPath } from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, isBoardPath, isSettingsPath, isVelocityPath } from "./factoriesRail";

interface FactoriesSidebarNavProps {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  canOpenSettings: boolean;
  permissionsLoading: boolean;
  /** Shows the Velocity rail link when the org has the `factory_velocity` experimental feature. */
  showVelocity: boolean;
}

function railLinkClassName(isCurrent: boolean) {
  return cn(factoriesRailControlClassName, isCurrent && "bg-sidebar-accent text-foreground");
}

function RailNavLink({
  to,
  label,
  Icon,
  testId,
  isCurrent,
}: {
  to: string;
  label: string;
  Icon: LucideIcon;
  testId: string;
  isCurrent: boolean;
}) {
  return (
    <Link
      to={to}
      aria-label={label}
      title={label}
      aria-current={isCurrent ? "page" : undefined}
      data-testid={testId}
      className={railLinkClassName(isCurrent)}
    >
      <Icon className="size-3.5" aria-hidden />
    </Link>
  );
}

/**
 * Icon rail under the workspace switcher: the line board, an optional
 * Velocity link, then settings. Velocity only shows when the organization
 * has the `factory_velocity` experimental feature enabled. Intakes and PR
 * feedback open from their listener rows on the board, so they do not need a
 * rail icon.
 */
export function FactoriesSidebarNav({
  organizationId,
  factoryKey,
  lineId,
  canOpenSettings,
  permissionsLoading,
  showVelocity,
}: FactoriesSidebarNavProps) {
  const { pathname } = useLocation();
  const boardHref = factoryHomePath(organizationId, factoryKey, lineId);
  const velocityHref = factoryVelocityPath(organizationId, factoryKey);
  const settingsHref = factorySettingsWorkspaceGeneralPath(organizationId, factoryKey);
  const boardCurrent = isBoardPath(pathname);
  const velocityCurrent = isVelocityPath(pathname);
  const settingsCurrent = isSettingsPath(pathname);

  return (
    <nav className="flex flex-col items-center gap-1 px-1.5" aria-label="Workspace" data-testid="factories-sidebar-nav">
      <RailNavLink to={boardHref} label="Board" Icon={Kanban} testId="factories-nav-board" isCurrent={boardCurrent} />
      {showVelocity ? (
        <RailNavLink
          to={velocityHref}
          label="Velocity"
          Icon={Gauge}
          testId="factories-nav-velocity"
          isCurrent={velocityCurrent}
        />
      ) : null}
      <PermissionTooltip
        allowed={canOpenSettings || permissionsLoading}
        message="You don't have permission to open workspace settings."
      >
        <Link
          to={canOpenSettings ? settingsHref : "#"}
          onClick={(event) => {
            if (!canOpenSettings) {
              event.preventDefault();
            }
          }}
          aria-label="Workspace settings"
          title="Workspace settings"
          aria-current={settingsCurrent ? "page" : undefined}
          data-testid="factories-workspace-settings-link"
          className={cn(railLinkClassName(settingsCurrent), !canOpenSettings && "pointer-events-none opacity-60")}
        >
          <Settings className="size-3.5" aria-hidden />
        </Link>
      </PermissionTooltip>
    </nav>
  );
}
