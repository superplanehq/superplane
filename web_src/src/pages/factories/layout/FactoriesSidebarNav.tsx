import { cn } from "@/lib/utils";
import { Gauge, Kanban, Settings } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link, useLocation } from "react-router";
import {
  factoryHomePath,
  factorySettingsSectionPath,
  factorySettingsWorkspaceGeneralPath,
  factoryVelocityPath,
} from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, isBoardPath, isSettingsPath, isVelocityPath } from "./factoriesRail";

interface FactoriesSidebarNavProps {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  canOpenSettings: boolean;
  permissionsLoading: boolean;
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
 * Icon rail under the workspace switcher: the line board, the Velocity link,
 * then settings. Intakes and PR feedback open from their listener rows on
 * the board, so they do not need a rail icon.
 */
export function FactoriesSidebarNav({ organizationId, factoryKey, lineId, canOpenSettings }: FactoriesSidebarNavProps) {
  const { pathname } = useLocation();
  const boardHref = factoryHomePath(organizationId, factoryKey, lineId);
  const velocityHref = factoryVelocityPath(organizationId, factoryKey);
  const settingsHref = canOpenSettings
    ? factorySettingsWorkspaceGeneralPath(organizationId, factoryKey)
    : factorySettingsSectionPath(organizationId, factoryKey, "account", "profile");
  const boardCurrent = isBoardPath(pathname);
  const velocityCurrent = isVelocityPath(pathname);
  const settingsCurrent = isSettingsPath(pathname);

  return (
    <nav className="flex flex-col items-center gap-1 px-1.5" aria-label="Workspace" data-testid="factories-sidebar-nav">
      <RailNavLink to={boardHref} label="Board" Icon={Kanban} testId="factories-nav-board" isCurrent={boardCurrent} />
      <RailNavLink
        to={velocityHref}
        label="Velocity"
        Icon={Gauge}
        testId="factories-nav-velocity"
        isCurrent={velocityCurrent}
      />
      <RailNavLink
        to={settingsHref}
        label="Workspace settings"
        Icon={Settings}
        testId="factories-workspace-settings-link"
        isCurrent={settingsCurrent}
      />
    </nav>
  );
}
