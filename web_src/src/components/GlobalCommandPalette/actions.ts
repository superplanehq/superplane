import {
  ArrowRightLeft,
  BookOpen,
  Boxes,
  CircleUser,
  Home,
  LogOut,
  MemoryStick,
  Palette,
  PanelTop,
  PlayCircle,
  Plug,
  Plus,
  Search,
  Settings,
  Shield,
} from "lucide-react";
import { ADMIN_LINKS, DOCS_URL, ORGANIZATION_SETTINGS_LINKS } from "./constants";
import type { CommandPage, PaletteAction, PalettePageAction } from "./types";

type RootPageActionParams = {
  accountInstallationAdmin: boolean;
  canReadCanvas: boolean;
  canvasId: string | null;
  currentCanvasName: string;
  organizationId: string | null;
  organizationName: string;
};

type RootActionParams = {
  accountEmail: string;
  canCreateCanvas: boolean;
  createCanvas: () => void;
  createCanvasPending: boolean;
  goTo: (href: string) => void;
  openExternal: (href: string) => void;
  organizationId: string | null;
  organizationName: string;
  shortcutModifier: string;
  signOut: () => void;
};

type CurrentCanvasActionParams = {
  canvasId: string | null;
  currentCanvasName: string;
  goTo: (href: string) => void;
  goToCurrentCanvasView: (view?: "dashboard" | "memory" | "runs") => void;
  organizationId: string | null;
};

export function buildRootPageActions({
  accountInstallationAdmin,
  canReadCanvas,
  canvasId,
  currentCanvasName,
  organizationId,
  organizationName,
}: RootPageActionParams): PalettePageAction[] {
  const actions: PalettePageAction[] = [
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
  ];

  if (accountInstallationAdmin) {
    actions.push({
      id: "admin-page",
      label: "Installation Admin",
      description: "Organizations, accounts, settings, and runner tasks",
      icon: Shield,
      page: "admin",
      keywords: ["admin", "installation", "accounts", "runner"],
    });
  }

  return actions;
}

export function buildRootActions({
  accountEmail,
  canCreateCanvas,
  createCanvas,
  createCanvasPending,
  goTo,
  openExternal,
  organizationId,
  organizationName,
  shortcutModifier,
  signOut,
}: RootActionParams): PaletteAction[] {
  return [
    {
      id: "new-canvas",
      label: createCanvasPending ? "Creating canvas..." : "New Canvas",
      description: organizationId ? `Create a blank canvas in ${organizationName}` : "Choose an organization first",
      icon: Plus,
      shortcut: `${shortcutModifier}/`,
      disabled: !canCreateCanvas || createCanvasPending,
      onSelect: () => createCanvas(),
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
    ...buildOrganizationRootActions(organizationId, organizationName, accountEmail, goTo),
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
      description: accountEmail,
      icon: LogOut,
      onSelect: signOut,
      keywords: ["logout", "account"],
    },
  ];
}

export function buildCurrentCanvasActions({
  canvasId,
  currentCanvasName,
  goTo,
  goToCurrentCanvasView,
  organizationId,
}: CurrentCanvasActionParams): PaletteAction[] {
  if (!organizationId || !canvasId) return [];

  return [
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
  ];
}

export function buildOrganizationSettingsActions({
  canAct,
  goTo,
  organizationId,
  usageEnabled,
}: {
  canAct: (resource: string, action: string) => boolean;
  goTo: (href: string) => void;
  organizationId: string | null;
  usageEnabled: boolean;
}): PaletteAction[] {
  return ORGANIZATION_SETTINGS_LINKS.filter((link) => link.id !== "usage" || usageEnabled).map((link) => ({
    id: link.id,
    label: link.label,
    description: link.description,
    icon: link.icon,
    disabled: link.permission ? !canAct(link.permission.resource, link.permission.action) : false,
    onSelect: () => organizationId && goTo(`/${organizationId}/${link.path}`),
  }));
}

export function buildAdminActions(goTo: (href: string) => void): PaletteAction[] {
  return ADMIN_LINKS.map((link) => ({
    id: link.id,
    label: link.label,
    description: link.description,
    icon: link.icon,
    onSelect: () => goTo(link.href),
  }));
}

function buildOrganizationRootActions(
  organizationId: string | null,
  organizationName: string,
  accountEmail: string,
  goTo: (href: string) => void,
): PaletteAction[] {
  if (!organizationId) return [];

  return [
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
      description: accountEmail,
      icon: CircleUser,
      onSelect: () => goTo(`/${organizationId}/settings/profile`),
      keywords: ["account", "profile", "user"],
    },
  ];
}

export function openPageAction(page: CommandPage, onOpenPage: (page: CommandPage) => void) {
  return () => onOpenPage(page);
}
