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
import { ArrowRightLeft, Check, Factory as FactoryIcon, Plus, Settings } from "lucide-react";
import { Link, useNavigate } from "react-router";
import { factoryDetailPath, factorySettingsPath } from "../lib/factoryPagePaths";

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
  const settingsHref = factory.id ? factorySettingsPath(organizationId, factory.id) : "#";

  return (
    <div className="flex items-center gap-0.5 px-2 pt-3 pb-1" data-testid="factories-workspace-switcher">
      <p
        className="flex-1 truncate rounded-md px-2.5 py-1.5 text-[13px] font-medium tracking-[-0.01em] text-foreground hover:bg-sidebar-accent"
        title={factory.name ?? undefined}
      >
        {factory.name}
      </p>
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
          data-testid="factories-workspace-settings-link"
          className={cn(
            "flex size-6 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-foreground",
            !canOpenSettings && "pointer-events-none opacity-60",
          )}
        >
          <Settings className="size-3.5" aria-hidden />
        </Link>
      </PermissionTooltip>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label="Switch workspace"
            className="flex size-6 items-center justify-center rounded-md text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
            data-testid="factories-workspace-switch"
          >
            <ArrowRightLeft className="size-3.5" aria-hidden />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-64">
          <DropdownMenuLabel>Switch workspace</DropdownMenuLabel>
          {factories.map((entry) => {
            const isCurrent = entry.id === factory.id;
            const targetHref = entry.id ? factoryDetailPath(organizationId, entry.id) : "";
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
                <FactoryIcon className="h-3.5 w-3.5" aria-hidden />
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
    </div>
  );
}
