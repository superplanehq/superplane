import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { Plus, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";

interface CanvasToolbarProps {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
}

export function CanvasToolbar({ searchQuery, setSearchQuery }: CanvasToolbarProps) {
  const organizationId = useOrganizationId();
  const navigate = useNavigate();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const canCreateCanvases = canAct("canvases", "create");
  const allowed = canCreateCanvases || permissionsLoading;

  const handleNewApp = () => {
    if (!organizationId || !canCreateCanvases) return;
    navigate(`/${organizationId}/apps/new`);
  };

  return (
    <div className="flex w-full flex-col gap-3 sm:flex-row sm:items-center">
      <PermissionTooltip allowed={allowed} message="You don't have permission to create canvases.">
        <Button
          type="button"
          onClick={handleNewApp}
          disabled={!canCreateCanvases || !organizationId}
          aria-label="Create new app"
        >
          <Plus className="h-4 w-4" />
          New App
        </Button>
      </PermissionTooltip>

      <div className="min-w-0 w-full sm:ml-auto sm:w-80">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" size={16} />
          <Input
            placeholder="Filter apps..."
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            className="pl-8"
          />
        </div>
      </div>
    </div>
  );
}
