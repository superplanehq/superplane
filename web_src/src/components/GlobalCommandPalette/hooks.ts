import type { AuthorizationPermission } from "@/api-client";
import { useMe } from "@/hooks/useMe";
import { useCallback, useEffect, useMemo } from "react";
import type { Dispatch, SetStateAction } from "react";
import { COMMAND_SHORTCUT } from "./constants";
import { subscribeToOpenCommandPalette } from "./controller";
import { isEditableTarget } from "./route";
import type { CommandPage } from "./types";

export function usePalettePermissions(organizationId: string | null) {
  const { data: me, isLoading } = useMe(true, organizationId);

  const permissionSet = useMemo(() => toPermissionSet(me?.permissions ?? []), [me?.permissions]);

  const canAct = useCallback(
    (resource: string, action: string) => {
      if (!organizationId) return false;
      if (isLoading) return false;
      return permissionSet.has(`${resource.toLowerCase()}:${action.toLowerCase()}`);
    },
    [isLoading, organizationId, permissionSet],
  );

  return { canAct, isLoading };
}

export function useCommandPaletteShortcuts({
  canvasId,
  organizationId,
  createCanvas,
  createCanvasDisabled,
  enabled,
  open,
  page,
  search,
  setOpen,
  setPage,
  setSearch,
}: {
  canvasId: string | null;
  organizationId: string | null;
  createCanvas: () => Promise<void>;
  createCanvasDisabled: boolean;
  enabled: boolean;
  open: boolean;
  page: CommandPage;
  search: string;
  setOpen: Dispatch<SetStateAction<boolean>>;
  setPage: Dispatch<SetStateAction<CommandPage>>;
  setSearch: Dispatch<SetStateAction<string>>;
}) {
  useEffect(() => {
    if (!enabled) {
      setOpen(false);
      setPage("root");
      setSearch("");
      return;
    }

    return subscribeToOpenCommandPalette(({ page = "root", search = "" }) => {
      setPage(page);
      setSearch(search);
      setOpen(true);
    });
  }, [enabled, setOpen, setPage, setSearch]);

  useEffect(() => {
    if (!enabled) return;

    const onKeyDown = (event: KeyboardEvent) => {
      const usesModifier = event.metaKey || event.ctrlKey;

      if (canToggleCommandPalette({ canvasId, event, open, organizationId, usesModifier })) {
        event.preventDefault();
        setOpen((prev) => !prev);
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
  }, [canvasId, organizationId, createCanvas, createCanvasDisabled, enabled, open, page, search, setOpen, setPage]);
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

function canToggleCommandPalette({
  canvasId,
  event,
  open,
  organizationId,
  usesModifier,
}: {
  canvasId: string | null;
  event: KeyboardEvent;
  open: boolean;
  organizationId: string | null;
  usesModifier: boolean;
}) {
  if (!usesModifier) return false;
  if (event.key !== "k") return false;
  if (canvasId) return false;
  if (!organizationId) return false;
  return open || !isEditableTarget(event.target);
}
