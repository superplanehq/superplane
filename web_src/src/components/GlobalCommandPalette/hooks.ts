import { meMe } from "@/api-client";
import type { AuthorizationPermission } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { Dispatch, SetStateAction } from "react";
import { COMMAND_SHORTCUT } from "./constants";
import { subscribeToOpenCommandPalette } from "./controller";
import { isEditableTarget } from "./route";
import type { CommandPage } from "./types";

export function usePalettePermissions(organizationId: string | null, enabled: boolean) {
  const { data: permissions = [], isLoading } = useQuery({
    queryKey: ["command-palette", "permissions", organizationId],
    queryFn: async () => {
      const response = await meMe(withOrganizationHeader({ organizationId, query: { includePermissions: true } }));
      return response.data?.user?.permissions || [];
    },
    enabled: enabled && !!organizationId,
    staleTime: 5 * 60 * 1000,
  });

  const permissionSet = useMemo(() => toPermissionSet(permissions), [permissions]);

  const canAct = useCallback(
    (resource: string, action: string) => {
      if (!organizationId) return false;
      if (isLoading) return true;
      return permissionSet.has(`${resource.toLowerCase()}:${action.toLowerCase()}`);
    },
    [isLoading, organizationId, permissionSet],
  );

  return { canAct, isLoading };
}

export function useShortcutModifierLabel() {
  const [modifier, setModifier] = useState("Ctrl+");

  useEffect(() => {
    const platform = window.navigator.platform.toLowerCase();
    setModifier(platform.includes("mac") ? "⌘" : "Ctrl+");
  }, []);

  return modifier;
}

export function useCommandPaletteShortcuts({
  createCanvas,
  createCanvasDisabled,
  open,
  page,
  search,
  setOpen,
  setPage,
  setSearch,
}: {
  createCanvas: () => Promise<void>;
  createCanvasDisabled: boolean;
  open: boolean;
  page: CommandPage;
  search: string;
  setOpen: Dispatch<SetStateAction<boolean>>;
  setPage: Dispatch<SetStateAction<CommandPage>>;
  setSearch: Dispatch<SetStateAction<string>>;
}) {
  useEffect(() => {
    return subscribeToOpenCommandPalette(({ page = "root", search = "" }) => {
      setPage(page);
      setSearch(search);
      setOpen(true);
    });
  }, [setOpen, setPage, setSearch]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const usesModifier = event.metaKey || event.ctrlKey;
      const key = event.key.toLowerCase();

      if (usesModifier && key === "k") {
        event.preventDefault();
        setOpen((current) => !current);
        return;
      }

      if (usesModifier && event.key === COMMAND_SHORTCUT && !isEditableTarget(event.target)) {
        if (createCanvasDisabled) return;
        event.preventDefault();
        void createCanvas();
        return;
      }

      if (open && event.key === "Backspace" && !search && page !== "root") {
        event.preventDefault();
        setPage("root");
      }
    };

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [createCanvas, createCanvasDisabled, open, page, search, setOpen, setPage]);
}

function toPermissionSet(permissions: AuthorizationPermission[]) {
  return new Set(
    permissions
      .map((permission) => {
        const resource = permission.resource?.toLowerCase();
        const action = permission.action?.toLowerCase();
        if (!resource || !action) return null;
        return `${resource}:${action}`;
      })
      .filter((value): value is string => !!value),
  );
}
