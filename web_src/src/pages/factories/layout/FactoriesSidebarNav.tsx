import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { ArrowRightFromLine, Gauge, Kanban, Plus, Settings } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Link, useLocation } from "react-router";
import { factoryHomePath, factorySettingsPath, factoryVelocityPath, workOrdersPath } from "../lib/factoryPagePaths";
import { factoriesRailControlClassName } from "./factoriesRail";

interface FactoriesSidebarNavProps {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  canOpenSettings: boolean;
  canCreateWorkOrder: boolean;
  permissionsLoading: boolean;
  onCreateWorkOrder: () => void;
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
 * Icon rail under the workspace switcher: incoming work, the line board,
 * velocity, settings, then create.
 */
export function FactoriesSidebarNav({
  organizationId,
  factoryKey,
  lineId,
  canOpenSettings,
  canCreateWorkOrder,
  permissionsLoading,
  onCreateWorkOrder,
}: FactoriesSidebarNavProps) {
  const { pathname } = useLocation();
  const intakeHref = workOrdersPath(organizationId, factoryKey);
  const boardHref = factoryHomePath(organizationId, factoryKey, lineId);
  const velocityHref = factoryVelocityPath(organizationId, factoryKey);
  const settingsHref = factorySettingsPath(organizationId, factoryKey);
  const intakeCurrent = isIntakePath(pathname);
  const boardCurrent = isBoardPath(pathname);
  const velocityCurrent = isVelocityPath(pathname);
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
      <RailNavLink to={boardHref} label="Board" Icon={Kanban} testId="factories-nav-board" isCurrent={boardCurrent} />
      <RailNavLink
        to={velocityHref}
        label="Velocity"
        Icon={Gauge}
        testId="factories-nav-velocity"
        isCurrent={velocityCurrent}
      />
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
      <PermissionTooltip
        allowed={canCreateWorkOrder || permissionsLoading}
        message="You don't have permission to create work orders."
      >
        <button
          type="button"
          onClick={() => {
            if (canCreateWorkOrder) {
              onCreateWorkOrder();
            }
          }}
          disabled={!canCreateWorkOrder}
          aria-label="Create work order"
          title="Create work order"
          data-testid="factories-sidebar-create-work-order"
          className={factoriesRailControlClassName}
        >
          <Plus className="size-3.5" aria-hidden />
        </button>
      </PermissionTooltip>
    </nav>
  );
}

export function isIntakePath(pathname: string): boolean {
  return pathname.includes("/work-orders") || /\/work-order\//.test(pathname);
}

export function isBoardPath(pathname: string): boolean {
  return pathname.includes("/lines");
}

export function isVelocityPath(pathname: string): boolean {
  return pathname.includes("/velocity");
}

export function isSettingsPath(pathname: string): boolean {
  return pathname.includes("/settings");
}
