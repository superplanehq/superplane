import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { ArrowRightFromLine, Kanban, MessageSquare, Settings } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link, useLocation } from "react-router";
import {
  factoryHomePath,
  factoryIntakePath,
  factoryPRFeedbackPath,
  factorySettingsPath,
  isIntakeSearchOpen,
  isPRFeedbackSearchOpen,
} from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, isBoardPath, isSettingsPath } from "./factoriesRail";

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
 * Icon rail under the workspace switcher: intake drawer, PR feedback,
 * the line board, then settings.
 */
export function FactoriesSidebarNav({
  organizationId,
  factoryKey,
  lineId,
  canOpenSettings,
  permissionsLoading,
}: FactoriesSidebarNavProps) {
  const { pathname, search } = useLocation();
  const boardHref = factoryHomePath(organizationId, factoryKey, lineId);
  const intakeOpen = isIntakeSearchOpen(search);
  const prFeedbackOpen = isPRFeedbackSearchOpen(search);
  const intakeHref = intakeOpen ? boardHref : factoryIntakePath(organizationId, factoryKey, lineId);
  const prFeedbackHref = prFeedbackOpen ? boardHref : factoryPRFeedbackPath(organizationId, factoryKey, lineId);
  const settingsHref = factorySettingsPath(organizationId, factoryKey);
  const intakeCurrent = intakeOpen;
  const prFeedbackCurrent = prFeedbackOpen;
  const boardCurrent = isBoardPath(pathname) && !intakeOpen && !prFeedbackOpen;
  const settingsCurrent = isSettingsPath(pathname);

  return (
    <nav className="flex flex-col items-center gap-1 px-1.5" aria-label="Workspace" data-testid="factories-sidebar-nav">
      <RailNavLink
        to={intakeHref}
        label="Intake"
        Icon={ArrowRightFromLine}
        testId="factories-nav-intake"
        isCurrent={intakeCurrent}
      />
      <RailNavLink
        to={prFeedbackHref}
        label="PR feedback"
        Icon={MessageSquare}
        testId="factories-nav-pr-feedback"
        isCurrent={prFeedbackCurrent}
      />
      <RailNavLink to={boardHref} label="Board" Icon={Kanban} testId="factories-nav-board" isCurrent={boardCurrent} />
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
