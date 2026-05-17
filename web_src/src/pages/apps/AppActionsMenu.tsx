import { useState } from "react";
import type { MouseEvent } from "react";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreVertical, Trash2 } from "lucide-react";
import type { AppsApp } from "@/lib/appsApi";
import { DeleteAppDialog } from "./DeleteAppDialog";

interface AppActionsMenuProps {
  app: AppsApp;
  organizationId: string;
  canUpdateApps: boolean;
  canDeleteApps: boolean;
  permissionsLoading: boolean;
}

export function AppActionsMenu({
  app,
  organizationId: _organizationId,
  canUpdateApps: _canUpdateApps,
  canDeleteApps,
  permissionsLoading,
}: AppActionsMenuProps) {
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);

  const canManage = canDeleteApps;

  const stopPropagation = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const openDelete = (event: MouseEvent<HTMLElement>) => {
    stopPropagation(event);
    setIsDeleteOpen(true);
  };

  if (!canManage && !permissionsLoading) {
    return (
      <PermissionTooltip allowed={permissionsLoading} message="You don't have permission to manage this app.">
        <button
          className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="App actions"
          disabled
          onClick={stopPropagation}
        >
          <MoreVertical className="h-4 w-4" />
        </button>
      </PermissionTooltip>
    );
  }

  return (
    <div onClick={stopPropagation}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" aria-label="App actions" onClick={stopPropagation}>
            <MoreVertical className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={openDelete} className="text-red-600 focus:text-red-600" disabled={!canDeleteApps}>
            <Trash2 className="h-4 w-4 mr-2" />
            Delete App
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <DeleteAppDialog app={app} isOpen={isDeleteOpen} onClose={() => setIsDeleteOpen(false)} />
    </div>
  );
}
