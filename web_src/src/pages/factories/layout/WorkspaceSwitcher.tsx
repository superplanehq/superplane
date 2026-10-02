import type { FactoriesFactory } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { cn } from "@/lib/utils";
import { Check, Plus, Settings, Triangle } from "lucide-react";
import { Link, useLocation, useNavigate } from "react-router";
import { factoryRouteSegment } from "../lib/factoryKeyResolution";
import { factorySettingsWorkspaceGeneralPath, pathAfterWorkspaceSwitch } from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, initialsForName } from "./factoriesRail";

interface WorkspaceSwitcherProps {
  organizationId: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
  canCreateFactory: boolean;
  canOpenSettings: boolean;
  permissionsLoading: boolean;
  onCreateFactory: () => void;
  /** Public board shows the workspace name. It does not list other workspaces. */
  infoOnly?: boolean;
}

export function WorkspaceSwitcher({
  organizationId,
  factory,
  factories,
  canCreateFactory,
  canOpenSettings,
  permissionsLoading,
  onCreateFactory,
  infoOnly = false,
}: WorkspaceSwitcherProps) {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const workspaceName = factory.name?.trim() || "Workspace";
  const currentRouteSegment = factoryRouteSegment(factory);

  return (
    <div className="flex flex-col items-center gap-1 px-1.5 pt-3 pb-1" data-testid="factories-workspace-switcher">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label={infoOnly ? workspaceName : `Switch workspace, ${workspaceName}`}
            title={workspaceName}
            className={cn(
              factoriesRailControlClassName,
              "bg-sidebar-accent text-[11px] font-medium tracking-[-0.01em] text-foreground",
            )}
            data-testid="factories-workspace-switch"
          >
            {initialsForName(workspaceName)}
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" side="right" className="w-72">
          {infoOnly ? (
            <WorkspaceInfoName name={workspaceName} />
          ) : (
            <WorkspaceSwitcherMenu
              organizationId={organizationId}
              factory={factory}
              factories={factories}
              canCreateFactory={canCreateFactory}
              canOpenSettings={canOpenSettings}
              permissionsLoading={permissionsLoading}
              onCreateFactory={onCreateFactory}
              onSwitch={(next) => {
                if (!currentRouteSegment || !factoryRouteSegment(next)) {
                  return;
                }
                navigate(
                  pathAfterWorkspaceSwitch({
                    pathname,
                    organizationId,
                    currentFactoryKey: currentRouteSegment,
                    nextFactory: next,
                  }),
                );
              }}
            />
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function WorkspaceInfoName({ name }: { name: string }) {
  return (
    <div className="px-3 py-2" data-testid="factories-workspace-info">
      <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
    </div>
  );
}

function WorkspaceSwitcherMenu({
  organizationId,
  factory,
  factories,
  canCreateFactory,
  canOpenSettings,
  permissionsLoading,
  onCreateFactory,
  onSwitch,
}: Omit<WorkspaceSwitcherProps, "infoOnly"> & { onSwitch: (next: FactoriesFactory) => void }) {
  const currentRouteSegment = factoryRouteSegment(factory);
  return (
    <>
      <DropdownMenuLabel>Switch workspace</DropdownMenuLabel>
      {factories.map((entry) => (
        <WorkspaceSwitcherRow
          key={entry.id}
          organizationId={organizationId}
          entry={entry}
          isCurrent={entry.id === factory.id}
          currentFactoryKey={currentRouteSegment}
          canOpenSettings={canOpenSettings}
          onSwitch={onSwitch}
        />
      ))}
      <DropdownMenuSeparator />
      <PermissionTooltip
        allowed={canCreateFactory || permissionsLoading}
        message="You don't have permission to create workspaces."
      >
        <DropdownMenuItem
          disabled={!canCreateFactory}
          onClick={() => {
            if (canCreateFactory) {
              onCreateFactory();
            }
          }}
          data-testid="factories-workspace-create"
        >
          <Plus className="h-3.5 w-3.5" aria-hidden />
          Create new workspace
        </DropdownMenuItem>
      </PermissionTooltip>
    </>
  );
}

function WorkspaceSwitcherRow({
  organizationId,
  entry,
  isCurrent,
  currentFactoryKey,
  canOpenSettings,
  onSwitch,
}: {
  organizationId: string;
  entry: FactoriesFactory;
  isCurrent: boolean;
  currentFactoryKey?: string;
  canOpenSettings: boolean;
  onSwitch: (next: FactoriesFactory) => void;
}) {
  const routeSegment = factoryRouteSegment(entry);
  const settingsHref = routeSegment ? factorySettingsWorkspaceGeneralPath(organizationId, routeSegment) : undefined;

  return (
    <div className="flex items-center gap-0.5">
      <DropdownMenuItem
        className="min-w-0 flex-1"
        onClick={() => {
          if (isCurrent || !routeSegment || !currentFactoryKey) {
            return;
          }
          onSwitch(entry);
        }}
        data-testid={`factories-workspace-option-${entry.id}`}
        aria-current={isCurrent ? "true" : undefined}
      >
        <Triangle className="h-3.5 w-3.5" aria-hidden />
        <span className="min-w-0 flex-1 truncate">{entry.name}</span>
        {isCurrent ? <Check className="h-3.5 w-3.5 shrink-0" aria-hidden /> : null}
      </DropdownMenuItem>
      {canOpenSettings && settingsHref ? (
        <Link
          to={settingsHref}
          aria-label={isCurrent ? "Workspace settings" : `Workspace settings, ${entry.name}`}
          title="Workspace settings"
          data-testid={isCurrent ? "factories-workspace-settings-link" : `factories-workspace-settings-${entry.id}`}
          className="mr-1 flex size-7 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          <Settings className="size-3.5" aria-hidden />
        </Link>
      ) : null}
    </div>
  );
}
