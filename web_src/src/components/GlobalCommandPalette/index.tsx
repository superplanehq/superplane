import { meMe } from "@/api-client";
import type { AuthorizationPermission, CanvasesCanvas } from "@/api-client";
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
import { useAccount } from "@/contexts/useAccount";
import { useCanvases, useCreateCanvas } from "@/hooks/useCanvasData";
import { useOrganization, useOrganizationUsage } from "@/hooks/useOrganizationData";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { isUsagePageForced } from "@/lib/env";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { cn } from "@/lib/utils";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRightLeft,
  BookOpen,
  Bot,
  Boxes,
  Building2,
  ChevronRight,
  CircleUser,
  FileText,
  Gauge,
  Home,
  Key,
  LogOut,
  MemoryStick,
  Network,
  PanelTop,
  Palette,
  PlayCircle,
  Plug,
  Plus,
  Search,
  Settings,
  Shield,
  Terminal,
  User,
  Users,
  type LucideIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";

type CommandPage = "root" | "organization-settings" | "canvas-settings" | "open-canvas" | "admin";

type PermissionCheck = {
  resource: string;
  action: string;
};

type PaletteAction = {
  id: string;
  label: string;
  description?: string;
  icon: LucideIcon;
  keywords?: string[];
  shortcut?: string;
  disabled?: boolean;
  onSelect: () => void;
};

type PalettePageAction = Omit<PaletteAction, "onSelect"> & {
  page: CommandPage;
};

const COMMAND_SHORTCUT = "/";
const DOCS_URL = "https://docs.superplane.com";

const PUBLIC_TOP_LEVEL_SEGMENTS = new Set(["", "admin", "create", "invite", "install", "login", "setup"]);

const ORGANIZATION_SETTINGS_LINKS: Array<{
  id: string;
  label: string;
  description: string;
  path: string;
  icon: LucideIcon;
  permission?: PermissionCheck;
}> = [
  {
    id: "general",
    label: "Settings",
    description: "Organization basics and identity",
    path: "settings/general",
    icon: Settings,
    permission: { resource: "org", action: "read" },
  },
  {
    id: "members",
    label: "Members",
    description: "Invite people and manage access",
    path: "settings/members",
    icon: User,
    permission: { resource: "members", action: "read" },
  },
  {
    id: "service-accounts",
    label: "Service Accounts",
    description: "Programmatic API access",
    path: "settings/service-accounts",
    icon: Bot,
    permission: { resource: "service_accounts", action: "read" },
  },
  {
    id: "groups",
    label: "Groups",
    description: "Organize members for permissions",
    path: "settings/groups",
    icon: Users,
    permission: { resource: "groups", action: "read" },
  },
  {
    id: "roles",
    label: "Roles",
    description: "Configure fine-grained access",
    path: "settings/roles",
    icon: Shield,
    permission: { resource: "roles", action: "read" },
  },
  {
    id: "integrations",
    label: "Integrations",
    description: "Connect external services",
    path: "settings/integrations",
    icon: Plug,
    permission: { resource: "integrations", action: "read" },
  },
  {
    id: "usage",
    label: "Usage",
    description: "Limits and tracked usage",
    path: "settings/billing",
    icon: Gauge,
    permission: { resource: "org", action: "read" },
  },
  {
    id: "secrets",
    label: "Secrets",
    description: "Encrypted values for workflows",
    path: "settings/secrets",
    icon: Key,
    permission: { resource: "secrets", action: "read" },
  },
  {
    id: "profile",
    label: "Profile",
    description: "Personal account settings",
    path: "settings/profile",
    icon: CircleUser,
  },
];

const ADMIN_LINKS: Array<{
  id: string;
  label: string;
  description: string;
  href: string;
  icon: LucideIcon;
}> = [
  {
    id: "organizations",
    label: "Organizations",
    description: "Review organizations in this installation",
    href: "/admin",
    icon: Building2,
  },
  {
    id: "accounts",
    label: "Accounts",
    description: "Manage accounts and installation admins",
    href: "/admin/accounts",
    icon: Users,
  },
  {
    id: "settings",
    label: "Settings",
    description: "Installation network and SMTP settings",
    href: "/admin/settings",
    icon: Network,
  },
  {
    id: "runner-tasks",
    label: "Runner Tasks",
    description: "Inspect runner task activity",
    href: "/admin/runner-tasks",
    icon: Terminal,
  },
];

export function GlobalCommandPalette() {
  const { account, loading: accountLoading } = useAccount();
  const location = useLocation();
  const navigate = useNavigate();
  const routeContext = useMemo(() => getRouteContext(location.pathname), [location.pathname]);
  const organizationId = routeContext.organizationId;
  const canvasId = routeContext.canvasId;

  const [open, setOpen] = useState(false);
  const [page, setPage] = useState<CommandPage>("root");
  const [search, setSearch] = useState("");
  const shortcutModifier = useShortcutModifierLabel();

  const { data: organization } = useOrganization(organizationId || "");
  const { data: usageStatus, error: usageError } = useOrganizationUsage(organizationId || "", !!organizationId);
  const { data: canvases = [], isLoading: canvasesLoading } = useCanvases(organizationId || "");
  const permissionState = usePalettePermissions(organizationId, !!account);
  const createCanvasMutation = useCreateCanvas(organizationId || "");

  const currentCanvas = useMemo(
    () => canvases.find((canvas) => canvas.metadata?.id === canvasId),
    [canvasId, canvases],
  );

  const usageEnabled = usageStatus?.enabled === true || !!usageError || isUsagePageForced();
  const canCreateCanvas = !!organizationId && permissionState.canAct("canvases", "create");
  const canReadCanvas = !!organizationId && permissionState.canAct("canvases", "read");
  const isCreateCanvasDisabled = !canCreateCanvas || createCanvasMutation.isPending;

  const closePalette = useCallback(() => {
    setOpen(false);
    setPage("root");
    setSearch("");
  }, []);

  const goTo = useCallback(
    (href: string) => {
      closePalette();
      navigate(href);
    },
    [closePalette, navigate],
  );

  const openPage = useCallback((nextPage: CommandPage) => {
    setPage(nextPage);
    setSearch("");
  }, []);

  const openExternal = useCallback(
    (href: string) => {
      closePalette();
      window.open(href, "_blank", "noopener,noreferrer");
    },
    [closePalette],
  );

  const signOut = useCallback(() => {
    closePalette();
    window.location.href = "/logout";
  }, [closePalette]);

  const createCanvas = useCallback(async () => {
    if (!organizationId || !canCreateCanvas || createCanvasMutation.isPending) return;

    try {
      const result = await createCanvasMutation.mutateAsync({
        name: generateCanvasName(),
        method: "ui",
      });
      const nextCanvasId = result?.data?.canvas?.metadata?.id;
      if (nextCanvasId) {
        closePalette();
        navigate(`/${organizationId}/canvases/${nextCanvasId}`);
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create canvas"));
    }
  }, [canCreateCanvas, closePalette, createCanvasMutation, navigate, organizationId]);

  const goToCurrentCanvasView = useCallback(
    (view?: "dashboard" | "memory" | "runs") => {
      if (!organizationId || !canvasId) return;
      const search = view ? `?view=${view}` : "";
      goTo(`/${organizationId}/canvases/${canvasId}${search}`);
    },
    [canvasId, goTo, organizationId],
  );

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
        if (isCreateCanvasDisabled) return;
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
  }, [createCanvas, isCreateCanvasDisabled, open, page, search]);

  useEffect(() => {
    if (!open) {
      setPage("root");
      setSearch("");
    }
  }, [open]);

  if (accountLoading || !account) {
    return null;
  }

  const organizationName = organization?.metadata?.name || "Current organization";
  const currentCanvasName = currentCanvas?.metadata?.name || "Current canvas";
  const rootPageActions: PalettePageAction[] = [
    {
      id: "open-canvas-page",
      label: "Open Canvas",
      description: organizationId ? "Search canvases in this organization" : "Choose an organization first",
      icon: Search,
      page: "open-canvas",
      disabled: !organizationId || !canReadCanvas,
      keywords: ["canvas", "project", "workflow", "search"],
    },
    {
      id: "organization-settings-page",
      label: "Organization Settings",
      description: organizationId ? organizationName : "Choose an organization first",
      icon: Settings,
      page: "organization-settings",
      disabled: !organizationId,
      keywords: ["members", "groups", "roles", "integrations", "secrets", "service accounts", "billing"],
    },
    {
      id: "canvas-settings-page",
      label: "Canvas Settings",
      description: canvasId ? currentCanvasName : "Pick a canvas to configure",
      icon: Palette,
      page: "canvas-settings",
      disabled: !organizationId || !canReadCanvas,
      keywords: ["canvas", "settings", "configuration"],
    },
    ...(account.installation_admin
      ? [
          {
            id: "admin-page",
            label: "Installation Admin",
            description: "Organizations, accounts, settings, and runner tasks",
            icon: Shield,
            page: "admin" as const,
            keywords: ["admin", "installation", "accounts", "runner"],
          },
        ]
      : []),
  ];

  const rootActions: PaletteAction[] = [
    {
      id: "new-canvas",
      label: createCanvasMutation.isPending ? "Creating canvas..." : "New Canvas",
      description: organizationId ? `Create a blank canvas in ${organizationName}` : "Choose an organization first",
      icon: Plus,
      shortcut: `${shortcutModifier}/`,
      disabled: isCreateCanvasDisabled,
      onSelect: () => void createCanvas(),
      keywords: ["new", "create", "canvas", "project", "workflow"],
    },
    {
      id: "new-organization",
      label: "New Organization",
      description: "Create another organization",
      icon: Plus,
      onSelect: () => goTo("/create"),
      keywords: ["new", "create", "organization", "workspace"],
    },
    ...(organizationId
      ? [
          {
            id: "go-home",
            label: "Canvases",
            description: organizationName,
            icon: Home,
            onSelect: () => goTo(`/${organizationId}`),
            keywords: ["home", "dashboard", "canvases", "projects"],
          },
          {
            id: "templates",
            label: "Templates",
            description: "Browse reusable canvases",
            icon: Boxes,
            onSelect: () => goTo(`/${organizationId}/templates`),
            keywords: ["template", "canvas"],
          },
          {
            id: "profile",
            label: "Profile",
            description: account.email,
            icon: CircleUser,
            onSelect: () => goTo(`/${organizationId}/settings/profile`),
            keywords: ["account", "profile", "user"],
          },
        ]
      : []),
    {
      id: "change-organization",
      label: "Change Organization",
      description: "Return to organization picker",
      icon: ArrowRightLeft,
      onSelect: () => goTo("/"),
      keywords: ["switch", "organization", "workspace"],
    },
    {
      id: "install",
      label: "Install GitHub App",
      description: "Connect source control to SuperPlane",
      icon: Plug,
      onSelect: () => goTo("/install"),
      keywords: ["github", "install", "app", "integration"],
    },
    {
      id: "docs",
      label: "Go to Docs",
      description: "Open product documentation in a new tab",
      icon: BookOpen,
      onSelect: () => openExternal(DOCS_URL),
      keywords: ["help", "documentation", "docs"],
    },
    {
      id: "sign-out",
      label: "Sign Out",
      description: account.email,
      icon: LogOut,
      onSelect: signOut,
      keywords: ["logout", "account"],
    },
  ];

  const currentCanvasActions: PaletteAction[] =
    organizationId && canvasId
      ? [
          {
            id: "current-canvas",
            label: "Canvas",
            description: currentCanvasName,
            icon: Palette,
            onSelect: () => goToCurrentCanvasView(),
            keywords: ["canvas", "workflow"],
          },
          {
            id: "current-canvas-console",
            label: "Console",
            description: currentCanvasName,
            icon: PanelTop,
            onSelect: () => goToCurrentCanvasView("dashboard"),
            keywords: ["dashboard", "console"],
          },
          {
            id: "current-canvas-runs",
            label: "Runs",
            description: currentCanvasName,
            icon: PlayCircle,
            onSelect: () => goToCurrentCanvasView("runs"),
            keywords: ["runs", "executions"],
          },
          {
            id: "current-canvas-memory",
            label: "Memory",
            description: currentCanvasName,
            icon: MemoryStick,
            onSelect: () => goToCurrentCanvasView("memory"),
            keywords: ["memory"],
          },
          {
            id: "current-canvas-settings",
            label: "Canvas Settings",
            description: currentCanvasName,
            icon: Settings,
            onSelect: () => goTo(`/${organizationId}/canvases/${canvasId}/settings`),
            keywords: ["canvas", "settings"],
          },
        ]
      : [];

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Command Palette"
      description="Search pages, actions, and utilities."
      className="top-[12vh] max-h-[min(760px,80vh)] w-[calc(100vw-2rem)] max-w-3xl translate-y-0 overflow-hidden rounded-xl border border-slate-200 bg-white p-0 shadow-2xl sm:top-[14vh]"
      showCloseButton={false}
    >
      <CommandInput
        value={search}
        onValueChange={setSearch}
        placeholder={page === "root" ? "What can we help with?" : pageTitle(page)}
        className="h-16 text-lg"
      />
      <CommandList className="max-h-[min(600px,calc(80vh-4rem))] scroll-py-2 px-3 py-3">
        <CommandEmpty>No commands found.</CommandEmpty>
        {page === "root" ? (
          <>
            <CommandGroup heading="Create">
              {rootActions.slice(0, 2).map((action) => (
                <ActionItem key={action.id} action={action} />
              ))}
            </CommandGroup>

            <CommandSeparator className="my-2" />

            {currentCanvasActions.length > 0 ? (
              <>
                <CommandGroup heading="Current Canvas">
                  {currentCanvasActions.map((action) => (
                    <ActionItem key={action.id} action={action} />
                  ))}
                </CommandGroup>
                <CommandSeparator className="my-2" />
              </>
            ) : null}

            <CommandGroup heading="Navigate">
              {rootPageActions.map((action) => (
                <PageItem key={action.id} action={action} onSelect={() => openPage(action.page)} />
              ))}
              {rootActions.slice(2, -2).map((action) => (
                <ActionItem key={action.id} action={action} />
              ))}
            </CommandGroup>

            <CommandSeparator className="my-2" />

            <CommandGroup heading="Help and Account">
              {rootActions.slice(-2).map((action) => (
                <ActionItem key={action.id} action={action} />
              ))}
            </CommandGroup>
          </>
        ) : null}

        {page === "organization-settings" ? (
          <NestedPage onBack={() => openPage("root")}>
            <CommandGroup heading={organizationName}>
              {ORGANIZATION_SETTINGS_LINKS.filter((link) => {
                if (link.id === "usage" && !usageEnabled) return false;
                return true;
              }).map((link) => {
                const disabled = link.permission
                  ? !permissionState.canAct(link.permission.resource, link.permission.action)
                  : false;
                return (
                  <ActionItem
                    key={link.id}
                    action={{
                      id: link.id,
                      label: link.label,
                      description: link.description,
                      icon: link.icon,
                      disabled,
                      onSelect: () => organizationId && goTo(`/${organizationId}/${link.path}`),
                    }}
                  />
                );
              })}
            </CommandGroup>
          </NestedPage>
        ) : null}

        {page === "canvas-settings" ? (
          <NestedPage onBack={() => openPage("root")}>
            <CommandGroup heading={canvasId ? "Current Canvas" : "Canvases"}>
              {canvasId ? (
                <ActionItem
                  action={{
                    id: "current-canvas-settings",
                    label: currentCanvasName,
                    description: "Open canvas settings",
                    icon: Settings,
                    onSelect: () => organizationId && goTo(`/${organizationId}/canvases/${canvasId}/settings`),
                  }}
                />
              ) : null}
              {canvasListActions({
                canvases,
                canvasesLoading,
                emptyLabel: "No canvases available.",
                onSelect: (canvas) => {
                  const id = canvas.metadata?.id;
                  if (organizationId && id) goTo(`/${organizationId}/canvases/${id}/settings`);
                },
                icon: Settings,
                description: "Open canvas settings",
              })}
            </CommandGroup>
          </NestedPage>
        ) : null}

        {page === "open-canvas" ? (
          <NestedPage onBack={() => openPage("root")}>
            <CommandGroup heading="Canvases">
              {canvasListActions({
                canvases,
                canvasesLoading,
                emptyLabel: "No canvases available.",
                onSelect: (canvas) => {
                  const id = canvas.metadata?.id;
                  if (organizationId && id) goTo(`/${organizationId}/canvases/${id}`);
                },
                icon: Palette,
                description: "Open canvas",
              })}
            </CommandGroup>
          </NestedPage>
        ) : null}

        {page === "admin" ? (
          <NestedPage onBack={() => openPage("root")}>
            <CommandGroup heading="Installation Admin">
              {ADMIN_LINKS.map((link) => (
                <ActionItem
                  key={link.id}
                  action={{
                    id: link.id,
                    label: link.label,
                    description: link.description,
                    icon: link.icon,
                    onSelect: () => goTo(link.href),
                  }}
                />
              ))}
            </CommandGroup>
          </NestedPage>
        ) : null}
      </CommandList>
    </CommandDialog>
  );
}

function ActionItem({ action }: { action: PaletteAction }) {
  const Icon = action.icon;
  const value = [action.label, action.description, ...(action.keywords || [])].filter(Boolean).join(" ");

  return (
    <CommandItem
      value={value}
      disabled={action.disabled}
      onSelect={action.onSelect}
      className={cn(
        "min-h-14 cursor-pointer rounded-lg border border-transparent px-3 py-2.5 data-[selected=true]:border-slate-200 data-[selected=true]:bg-slate-100",
        action.disabled && "cursor-not-allowed",
      )}
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-600">
        <Icon className="h-4 w-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium text-slate-900">{action.label}</span>
        {action.description ? (
          <span className="block truncate text-xs text-slate-500">{action.description}</span>
        ) : null}
      </span>
      {action.shortcut ? <CommandShortcut>{action.shortcut}</CommandShortcut> : null}
    </CommandItem>
  );
}

function PageItem({ action, onSelect }: { action: PalettePageAction; onSelect: () => void }) {
  const Icon = action.icon;
  const value = [action.label, action.description, ...(action.keywords || [])].filter(Boolean).join(" ");

  return (
    <CommandItem
      value={value}
      disabled={action.disabled}
      onSelect={onSelect}
      className={cn(
        "min-h-14 cursor-pointer rounded-lg border border-transparent px-3 py-2.5 data-[selected=true]:border-slate-200 data-[selected=true]:bg-slate-100",
        action.disabled && "cursor-not-allowed",
      )}
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-600">
        <Icon className="h-4 w-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium text-slate-900">{action.label}</span>
        {action.description ? (
          <span className="block truncate text-xs text-slate-500">{action.description}</span>
        ) : null}
      </span>
      <ChevronRight className="h-4 w-4 text-slate-400" />
    </CommandItem>
  );
}

function NestedPage({ children, onBack }: { children: ReactNode; onBack: () => void }) {
  return (
    <>
      <CommandGroup>
        <CommandItem
          value="back return previous"
          onSelect={onBack}
          className="min-h-11 cursor-pointer rounded-lg px-3 py-2 data-[selected=true]:bg-slate-100"
        >
          <ArrowLeft className="h-4 w-4 text-slate-500" />
          <span className="text-sm font-medium text-slate-700">Back to commands</span>
        </CommandItem>
      </CommandGroup>
      <CommandSeparator className="my-2" />
      {children}
    </>
  );
}

function canvasListActions({
  canvases,
  canvasesLoading,
  emptyLabel,
  onSelect,
  icon,
  description,
}: {
  canvases: CanvasesCanvas[];
  canvasesLoading: boolean;
  emptyLabel: string;
  onSelect: (canvas: CanvasesCanvas) => void;
  icon: LucideIcon;
  description: string;
}) {
  if (canvasesLoading) {
    return (
      <CommandItem disabled value="loading canvases" className="min-h-12 rounded-lg px-3 py-2.5">
        <FileText className="h-4 w-4 text-slate-400" />
        <span className="text-sm text-slate-500">Loading canvases...</span>
      </CommandItem>
    );
  }

  if (canvases.length === 0) {
    return (
      <CommandItem disabled value="no canvases" className="min-h-12 rounded-lg px-3 py-2.5">
        <FileText className="h-4 w-4 text-slate-400" />
        <span className="text-sm text-slate-500">{emptyLabel}</span>
      </CommandItem>
    );
  }

  return canvases
    .filter((canvas) => canvas.metadata?.id)
    .map((canvas) => (
      <ActionItem
        key={canvas.metadata?.id}
        action={{
          id: canvas.metadata?.id || "",
          label: canvas.metadata?.name || "Untitled canvas",
          description,
          icon,
          onSelect: () => onSelect(canvas),
          keywords: [canvas.metadata?.description || ""],
        }}
      />
    ));
}

function usePalettePermissions(organizationId: string | null, enabled: boolean) {
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

function getRouteContext(pathname: string): { organizationId: string | null; canvasId: string | null } {
  const segments = pathname.split("/").filter(Boolean);
  const firstSegment = segments[0] || "";
  const organizationId = PUBLIC_TOP_LEVEL_SEGMENTS.has(firstSegment) ? null : firstSegment;
  const canvasIndex = segments.indexOf("canvases");
  const canvasId = canvasIndex >= 0 ? segments[canvasIndex + 1] || null : null;
  return { organizationId, canvasId };
}

function pageTitle(page: CommandPage): string {
  switch (page) {
    case "organization-settings":
      return "Search organization settings";
    case "canvas-settings":
      return "Search canvas settings";
    case "open-canvas":
      return "Search canvases";
    case "admin":
      return "Search admin pages";
    default:
      return "What can we help with?";
  }
}

function useShortcutModifierLabel() {
  const [modifier, setModifier] = useState("Ctrl+");

  useEffect(() => {
    const platform = window.navigator.platform.toLowerCase();
    setModifier(platform.includes("mac") ? "⌘" : "Ctrl+");
  }, []);

  return modifier;
}

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tagName = target.tagName.toLowerCase();
  return tagName === "input" || tagName === "textarea" || tagName === "select";
}
