import { useCallback, useEffect, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Building2, FileCode, Home, LayoutGrid, Plus, Settings, Shield } from "lucide-react";

import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "@/components/ui/command";
import { useCanvases } from "@/hooks/useCanvasData";
import { useAccount } from "@/contexts/AccountContext";
import { usePermissions } from "@/contexts/PermissionsContext";
import { analytics } from "@/lib/analytics";

export interface CommandPaletteProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type ActionGroup = "navigation" | "canvas" | "action" | "account";

const MAX_CANVASES = 8;

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const navigate = useNavigate();
  const { organizationId } = useParams<{ organizationId: string }>();
  const orgId = organizationId ?? "";
  const { account } = useAccount();
  const { canAct } = usePermissions();

  const { data: canvases = [] } = useCanvases(orgId);

  useEffect(() => {
    if (open && orgId) {
      analytics.commandPaletteOpen(orgId);
    }
  }, [open, orgId]);

  const run = useCallback(
    (actionId: string, group: ActionGroup, fn: () => void) => {
      if (orgId) analytics.commandPaletteAction(actionId, group, orgId);
      onOpenChange(false);
      fn();
    },
    [onOpenChange, orgId],
  );

  const recentCanvases = useMemo(() => {
    return [...canvases]
      .sort((a, b) => {
        const aDate = a.metadata?.updatedAt ?? a.metadata?.createdAt ?? "";
        const bDate = b.metadata?.updatedAt ?? b.metadata?.createdAt ?? "";
        return bDate.localeCompare(aDate);
      })
      .slice(0, MAX_CANVASES);
  }, [canvases]);

  const canCreate = canAct("canvases", "create");
  const isInstallationAdmin = !!account?.installation_admin;

  return (
    <CommandDialog open={open} onOpenChange={onOpenChange} title="SuperPlane Command Palette">
      <CommandInput placeholder="Search canvases or run an action..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        <CommandGroup heading="Navigation">
          <CommandItem
            value="navigation home canvases"
            onSelect={() => run("navigate_home", "navigation", () => navigate(`/${orgId}`))}
          >
            <Home />
            Home
            <CommandShortcut>G H</CommandShortcut>
          </CommandItem>
          <CommandItem
            value="navigation templates"
            onSelect={() => run("navigate_templates", "navigation", () => navigate(`/${orgId}/templates`))}
          >
            <FileCode />
            Templates
          </CommandItem>
          <CommandItem
            value="navigation organization settings"
            onSelect={() => run("navigate_org_settings", "navigation", () => navigate(`/${orgId}/settings`))}
          >
            <Settings />
            Organization settings
          </CommandItem>
          <CommandItem
            value="navigation switch organization"
            onSelect={() => run("switch_org", "account", () => navigate(`/`))}
          >
            <Building2 />
            Switch organization
          </CommandItem>
          {isInstallationAdmin && (
            <CommandItem
              value="navigation admin dashboard"
              onSelect={() => run("navigate_admin", "navigation", () => navigate(`/admin`))}
            >
              <Shield />
              Admin dashboard
            </CommandItem>
          )}
        </CommandGroup>

        {canCreate && (
          <>
            <CommandSeparator />
            <CommandGroup heading="Actions">
              <CommandItem
                value="action create canvas new"
                onSelect={() => run("create_canvas", "action", () => navigate(`/${orgId}/canvases/new`))}
              >
                <Plus />
                Create canvas
              </CommandItem>
            </CommandGroup>
          </>
        )}

        {recentCanvases.length > 0 && (
          <>
            <CommandSeparator />
            <CommandGroup heading="Canvases">
              {recentCanvases.map((canvas) => {
                const id = canvas.metadata?.id;
                const name = canvas.metadata?.name;
                if (!id || !name) return null;
                const description = canvas.metadata?.description ?? "";
                return (
                  <CommandItem
                    key={id}
                    value={`canvas ${name} ${description} ${id}`}
                    onSelect={() => run("open_canvas", "canvas", () => navigate(`/${orgId}/canvases/${id}`))}
                  >
                    <LayoutGrid />
                    <span className="truncate">{name}</span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
          </>
        )}
      </CommandList>
    </CommandDialog>
  );
}
