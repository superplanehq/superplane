import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { usePermissions } from "@/contexts/usePermissions";
import { Plus, Search } from "lucide-react";
import { useState } from "react";
import { CreateAppModal } from "./CreateAppModal";
import { useCreateApp } from "./useCreateApp";

interface CanvasToolbarProps {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  defaultAppName: string;
}

export function CanvasToolbar({ searchQuery, setSearchQuery, defaultAppName }: CanvasToolbarProps) {
  const [isCreateAppModalOpen, setIsCreateAppModalOpen] = useState(false);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { createApp, isSaving: isCreateAppSaving } = useCreateApp({
    onCreated: () => setIsCreateAppModalOpen(false),
  });

  const canCreateCanvases = canAct("canvases", "create");
  const allowed = canCreateCanvases || permissionsLoading;

  return (
    <>
      <div className="flex w-full flex-col gap-3 sm:flex-row sm:items-center">
        <PermissionTooltip allowed={allowed} message="You don't have permission to create canvases.">
          <Button
            type="button"
            onClick={() => setIsCreateAppModalOpen(true)}
            disabled={!canCreateCanvases}
            aria-label="Create new app"
          >
            <Plus className="h-4 w-4" />
            New App
          </Button>
        </PermissionTooltip>

        <div className="min-w-0 w-full sm:ml-auto sm:w-80">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
            <Input
              placeholder="Filter apps..."
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              className="pl-10"
            />
          </div>
        </div>
      </div>

      <CreateAppModal
        open={isCreateAppModalOpen}
        defaultName={defaultAppName}
        isSaving={isCreateAppSaving}
        onClose={() => setIsCreateAppModalOpen(false)}
        onSave={createApp}
      />
    </>
  );
}
