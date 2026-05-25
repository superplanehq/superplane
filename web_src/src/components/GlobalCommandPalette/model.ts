import type { CanvasesCanvas } from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { useCanvases, useCreateCanvas } from "@/hooks/useCanvasData";
import { useOrganization, useOrganizationUsage } from "@/hooks/useOrganizationData";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { isUsagePageForced } from "@/lib/env";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import {
  buildAdminActions,
  buildCurrentCanvasActions,
  buildOrganizationSettingsActions,
  buildRootActions,
  buildRootPageActions,
} from "./actions";
import { buildCanvasNodeSearchActions, useCanvasNodeSearchProvider } from "./canvasNodeSearchStore";
import { useCommandPaletteShortcuts, usePalettePermissions, useShortcutModifierLabel } from "./hooks";
import { getRouteContext } from "./route";
import type { CanvasCommandListProps, CommandPage, PaletteAction, PalettePageAction } from "./types";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { Dispatch, SetStateAction } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import type { NavigateFunction } from "react-router-dom";

export type CommandPaletteModel = {
  adminActions: PaletteAction[];
  canvasId: string | null;
  canvasListProps: CanvasCommandListProps;
  canvasNodeSearchActions: PaletteAction[];
  currentCanvasActions: PaletteAction[];
  currentCanvasName: string;
  open: boolean;
  organizationName: string;
  page: CommandPage;
  rootActions: PaletteAction[];
  rootPageActions: PalettePageAction[];
  search: string;
  setOpen: Dispatch<SetStateAction<boolean>>;
  setPage: Dispatch<SetStateAction<CommandPage>>;
  setSearch: Dispatch<SetStateAction<string>>;
  settingsActions: PaletteAction[];
};

export function useCommandPaletteModel(): CommandPaletteModel | null {
  const { account, loading } = useAccount();
  const location = useLocation();
  const navigate = useNavigate();
  const route = useMemo(() => getRouteContext(location.pathname), [location.pathname]);
  const [open, setOpen] = useState(false);
  const [page, setPage] = useState<CommandPage>("root");
  const [search, setSearch] = useState("");
  const shortcutModifier = useShortcutModifierLabel();
  const canvasNodeSearchProvider = useCanvasNodeSearchProvider();
  const data = useCommandPaletteData(route.organizationId, route.canvasId, !!account);
  const closePalette = useClosePalette(setOpen, setPage, setSearch);
  const navigation = usePaletteNavigation(route.organizationId, route.canvasId, closePalette, navigate);
  const createCanvas = useCreateCanvasCommand(data, closePalette, navigate, route.organizationId);

  useCommandPaletteShortcuts({
    createCanvas,
    createCanvasDisabled: data.createCanvasDisabled,
    open,
    page,
    search,
    setOpen,
    setPage,
  });

  useEffect(() => {
    if (!open) closePalette();
  }, [closePalette, open]);

  if (loading || !account) return null;

  return buildModel({
    accountEmail: account.email,
    accountInstallationAdmin: account.installation_admin,
    canvasId: route.canvasId,
    closePalette,
    createCanvas,
    canvasNodeSearchProvider,
    data,
    navigation,
    open,
    organizationId: route.organizationId,
    page,
    search,
    setOpen,
    setPage,
    setSearch,
    shortcutModifier,
  });
}

type PaletteData = {
  canCreateCanvas: boolean;
  canReadCanvas: boolean;
  canvases: CanvasesCanvas[];
  canvasesLoading: boolean;
  createCanvasDisabled: boolean;
  createCanvasMutation: ReturnType<typeof useCreateCanvas>;
  currentCanvasName: string;
  organizationName: string;
  permissionState: ReturnType<typeof usePalettePermissions>;
  usageEnabled: boolean;
};

function useCommandPaletteData(
  organizationId: string | null,
  canvasId: string | null,
  hasAccount: boolean,
): PaletteData {
  const queryOrganizationId = organizationId ?? "";
  const hasOrganization = organizationId !== null;
  const { data: organization } = useOrganization(queryOrganizationId);
  const { data: usageStatus, error: usageError } = useOrganizationUsage(queryOrganizationId, hasOrganization);
  const { data: canvases = [], isLoading: canvasesLoading } = useCanvases(queryOrganizationId);
  const permissionState = usePalettePermissions(organizationId, hasAccount);
  const createCanvasMutation = useCreateCanvas(queryOrganizationId);
  const currentCanvas = canvases.find((canvas) => canvas.metadata?.id === canvasId);
  const canCreateCanvas = canUsePermission(hasOrganization, permissionState.canAct, "canvases", "create");
  const canReadCanvas = canUsePermission(hasOrganization, permissionState.canAct, "canvases", "read");

  return {
    canCreateCanvas,
    canReadCanvas,
    canvases,
    canvasesLoading,
    createCanvasDisabled: !canCreateCanvas || createCanvasMutation.isPending,
    createCanvasMutation,
    currentCanvasName: currentCanvasNameFor(currentCanvas),
    organizationName: organization?.metadata?.name ?? "Current organization",
    permissionState,
    usageEnabled: isUsageEnabled(usageStatus?.enabled === true, usageError),
  };
}

function canUsePermission(
  hasOrganization: boolean,
  canAct: (resource: string, action: string) => boolean,
  resource: string,
  action: string,
) {
  if (!hasOrganization) return false;
  return canAct(resource, action);
}

function currentCanvasNameFor(canvas: CanvasesCanvas | undefined) {
  return canvas?.metadata?.name ?? "Current canvas";
}

function isUsageEnabled(enabled: boolean, error: unknown) {
  return enabled || !!error || isUsagePageForced();
}

function useClosePalette(
  setOpen: Dispatch<SetStateAction<boolean>>,
  setPage: Dispatch<SetStateAction<CommandPage>>,
  setSearch: Dispatch<SetStateAction<string>>,
) {
  return useCallback(() => {
    setOpen(false);
    setPage("root");
    setSearch("");
  }, [setOpen, setPage, setSearch]);
}

function usePaletteNavigation(
  organizationId: string | null,
  canvasId: string | null,
  closePalette: () => void,
  navigate: NavigateFunction,
) {
  const goTo = useCallback(
    (href: string) => {
      closePalette();
      navigate(href);
    },
    [closePalette, navigate],
  );

  const openExternal = useCallback(
    (href: string) => {
      closePalette();
      window.open(href, "_blank", "noopener,noreferrer");
    },
    [closePalette],
  );

  const goToCurrentCanvasView = useCallback(
    (view?: "dashboard" | "memory" | "runs") => {
      if (!organizationId || !canvasId) return;
      goTo(`/${organizationId}/canvases/${canvasId}${view ? `?view=${view}` : ""}`);
    },
    [canvasId, goTo, organizationId],
  );

  return { goTo, goToCurrentCanvasView, openExternal };
}

function useCreateCanvasCommand(
  data: PaletteData,
  closePalette: () => void,
  navigate: NavigateFunction,
  organizationId: string | null,
) {
  return useCallback(async () => {
    if (!organizationId || !data.canCreateCanvas || data.createCanvasMutation.isPending) return;

    try {
      const result = await data.createCanvasMutation.mutateAsync({ name: generateCanvasName(), method: "ui" });
      const nextCanvasId = result?.data?.canvas?.metadata?.id;
      if (!nextCanvasId) return;
      closePalette();
      navigate(`/${organizationId}/canvases/${nextCanvasId}`);
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create canvas"));
    }
  }, [closePalette, data.canCreateCanvas, data.createCanvasMutation, navigate, organizationId]);
}

function buildModel({
  accountEmail,
  accountInstallationAdmin,
  canvasId,
  closePalette,
  createCanvas,
  canvasNodeSearchProvider,
  data,
  navigation,
  open,
  organizationId,
  page,
  search,
  setOpen,
  setPage,
  setSearch,
  shortcutModifier,
}: {
  accountEmail: string;
  accountInstallationAdmin: boolean;
  canvasId: string | null;
  closePalette: () => void;
  createCanvas: () => Promise<void>;
  canvasNodeSearchProvider: ReturnType<typeof useCanvasNodeSearchProvider>;
  data: PaletteData;
  navigation: ReturnType<typeof usePaletteNavigation>;
  open: boolean;
  organizationId: string | null;
  page: CommandPage;
  search: string;
  setOpen: Dispatch<SetStateAction<boolean>>;
  setPage: Dispatch<SetStateAction<CommandPage>>;
  setSearch: Dispatch<SetStateAction<string>>;
  shortcutModifier: string;
}): CommandPaletteModel {
  return {
    adminActions: buildAdminActions(navigation.goTo),
    canvasId,
    canvasListProps: {
      canvases: data.canvases,
      canvasesLoading: data.canvasesLoading,
      goTo: navigation.goTo,
      organizationId,
    },
    canvasNodeSearchActions: buildCanvasNodeSearchActions({
      closePalette,
      provider: canvasNodeSearchProvider,
      query: search,
    }),
    currentCanvasActions: buildCurrentCanvasActions({
      canvasId,
      currentCanvasName: data.currentCanvasName,
      goTo: navigation.goTo,
      goToCurrentCanvasView: navigation.goToCurrentCanvasView,
      organizationId,
    }),
    currentCanvasName: data.currentCanvasName,
    open,
    organizationName: data.organizationName,
    page,
    rootActions: buildRootActions({
      accountEmail,
      canCreateCanvas: data.canCreateCanvas,
      createCanvas,
      createCanvasPending: data.createCanvasMutation.isPending,
      goTo: navigation.goTo,
      openExternal: navigation.openExternal,
      organizationId,
      organizationName: data.organizationName,
      shortcutModifier,
      signOut: () => {
        closePalette();
        window.location.href = "/logout";
      },
    }),
    rootPageActions: buildRootPageActions({
      accountInstallationAdmin,
      canReadCanvas: data.canReadCanvas,
      canvasId,
      currentCanvasName: data.currentCanvasName,
      hasCanvasNodeSearch: !!canvasId && !!canvasNodeSearchProvider,
      organizationId,
      organizationName: data.organizationName,
    }),
    search,
    setOpen,
    setPage,
    setSearch,
    settingsActions: buildOrganizationSettingsActions({
      canAct: data.permissionState.canAct,
      goTo: navigation.goTo,
      organizationId,
      usageEnabled: data.usageEnabled,
    }),
  };
}
