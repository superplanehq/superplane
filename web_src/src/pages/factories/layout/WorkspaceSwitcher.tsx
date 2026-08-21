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
import { Link, useNavigate } from "react-router";
import { factoryDetailPath, factorySettingsPath } from "../lib/factoryPagePaths";
import { factoriesRailControlClassName, initialsForName } from "./factoriesRail";

interface WorkspaceSwitcherProps {
  organizationId: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
  canOpenSettings: boolean;
  canCreateFactory: boolean;
  permissionsLoading: boolean;
  onCreateFactory: () => void;
}

export function WorkspaceSwitcher({
  organizationId,
  factory,
  factories,
  canOpenSettings,
  canCreateFactory,
  permissionsLoading,
  onCreateFactory,
}: WorkspaceSwitcherProps) {
  const navigate = useNavigate();
  const settingsHref = factory.key ? factorySettingsPath(organizationId, factory.key) : "#";
  const workspaceName = factory.name?.trim() || "Workspace";

  return (
    <div className="flex flex-col items-center gap-1 px-1.5 pt-3 pb-1" data-testid="factories-workspace-switcher">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label={`Switch workspace, ${workspaceName}`}
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
        <DropdownMenuContent align="start" side="right" className="w-64">
          <DropdownMenuLabel>Switch workspace</DropdownMenuLabel>
          {factories.map((entry) => {
            const isCurrent = entry.id === factory.id;
            const targetHref = entry.key ? factoryDetailPath(organizationId, entry.key) : "";
            return (
              <DropdownMenuItem
                key={entry.id}
                onClick={() => {
                  if (!isCurrent && targetHref) {
                    navigate(targetHref);
                  }
                }}
                data-testid={`factories-workspace-option-${entry.id}`}
              >
                <Triangle className="h-3.5 w-3.5" aria-hidden />
                <span className="truncate">{entry.name}</span>
                {isCurrent ? <Check className="ml-auto h-3.5 w-3.5" aria-hidden /> : null}
              </DropdownMenuItem>
            );
          })}
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
        </DropdownMenuContent>
      </DropdownMenu>
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
          data-testid="factories-workspace-settings-link"
          className={cn(factoriesRailControlClassName, !canOpenSettings && "pointer-events-none opacity-60")}
        >
          <Settings className="size-3.5" aria-hidden />
        </Link>
      </PermissionTooltip>
    </div>
  );
}
