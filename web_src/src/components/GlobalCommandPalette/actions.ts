import {
  ArrowRightLeft,
  BookOpen,
  CircleUser,
  LogOut,
  MemoryStick,
  Palette,
  PanelTop,
  PlayCircle,
  Plus,
  Settings,
  Sparkles,
} from "lucide-react";
import type { CanvasToolSidebarTab } from "@/components/CanvasToolSidebar/events";
import { ADMIN_LINKS, DOCS_URL, ORGANIZATION_SETTINGS_LINKS } from "./constants";
import { appSettingsPath } from "@/lib/appPaths";
import type { PaletteAction } from "./types";

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
  agentEnabled: boolean;
  canUpdateCanvas: boolean;
  canvasId: string | null;
  currentCanvasName: string;
  goTo: (href: string) => void;
  goToCurrentCanvasView: (view?: "console" | "memory") => void;
  openCurrentCanvasToolTab: (tab: CanvasToolSidebarTab) => void;
  organizationId: string | null;
  showToolTabCommands: boolean;
};

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
      label: createCanvasPending ? "Creating app..." : "New App",
      description: organizationId ? `Create a blank app in ${organizationName}` : "Choose an organization first",
      icon: Plus,
      shortcut: `${shortcutModifier}/`,
      disabled: !canCreateCanvas || createCanvasPending,
      onSelect: () => createCanvas(),
      keywords: ["new", "create", "canvas", "project", "workflow"],
    },
    ...buildOrganizationRootActions(organizationId, accountEmail, goTo),
    {
      id: "change-organization",
      label: "Change Organization",
      description: "Return to organization picker",
      icon: ArrowRightLeft,
      onSelect: () => goTo("/?select=true"),
      keywords: ["switch", "organization", "workspace"],
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
  agentEnabled,
  canUpdateCanvas,
  canvasId,
  currentCanvasName,
  goTo,
  goToCurrentCanvasView,
  openCurrentCanvasToolTab,
  organizationId,
  showToolTabCommands,
}: CurrentCanvasActionParams): PaletteAction[] {
  if (!organizationId || !canvasId) return [];

  const actions: PaletteAction[] = [
    {
      id: "current-canvas",
      label: "App",
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
      onSelect: () => goToCurrentCanvasView("console"),
      keywords: ["console"],
    },
    {
      id: "current-canvas-runs",
      label: "Runs",
      description: currentCanvasName,
      icon: PlayCircle,
      onSelect: () => goToCurrentCanvasView(),
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
      label: "App Settings",
      description: currentCanvasName,
      icon: Settings,
      disabled: !canUpdateCanvas,
      onSelect: () => organizationId && canvasId && goTo(appSettingsPath(organizationId, canvasId)),
      keywords: ["canvas", "settings"],
    },
  ];

  if (agentEnabled && showToolTabCommands) {
    actions.splice(1, 0, {
      id: "current-canvas-agent",
      label: "Agent",
      description: currentCanvasName,
      icon: Sparkles,
      onSelect: () => openCurrentCanvasToolTab("agent"),
      keywords: ["agent", "assistant", "build", "chat"],
    });
  }

  return actions;
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
  accountEmail: string,
  goTo: (href: string) => void,
): PaletteAction[] {
  if (!organizationId) return [];

  return [
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
