import { AgentSidebar } from "@/components/AgentSidebar";
import { useAgentState } from "@/components/AgentSidebar/useAgentState";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import {
  CANVAS_FOLDER_COLORS,
  DEFAULT_CANVAS_FOLDER_COLOR,
  useCanvases,
  useCanvas,
  useCanvasFolders,
  type CanvasFolderColor,
} from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { Search, Sparkles } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

type AppTab = "dashboard" | "canvas" | "runs" | "data";

type PrototypeCanvas = {
  metadata?: {
    id?: string;
    name?: string;
    liveVersionId?: string;
    folderId?: string;
  };
};

const APP_TABS: Array<{ id: AppTab; label: string }> = [
  { id: "dashboard", label: "Dashboard" },
  { id: "canvas", label: "Canvas" },
  { id: "runs", label: "Runs" },
  { id: "data", label: "Data" },
];

const RECENT_APPS_STORAGE_KEY_PREFIX = "appPrototypeRecentlyOpenedApps";
const MAX_RECENT_APPS = 20;
const APP_SWITCHER_FOLDER_DOT_CLASSES: Record<CanvasFolderColor, string> = {
  blue: "bg-blue-100",
  green: "bg-green-100",
  purple: "bg-purple-100",
  yellow: "bg-yellow-100",
  slate: "bg-slate-100",
  orange: "bg-orange-100",
};

export function AppPrototypePage() {
  const navigate = useNavigate();
  const { organizationId = "", canvasId = "" } = useParams<{ organizationId: string; canvasId: string }>();
  const [activeTab, setActiveTab] = useState<AppTab>("canvas");
  const [isSwitcherOpen, setIsSwitcherOpen] = useState(false);
  const [recentAppIds, setRecentAppIds] = useState<string[]>(() => readRecentAppIds(organizationId));

  const { data: canvasData, isLoading: isCanvasLoading } = useCanvas(organizationId, canvasId);
  const { data: appsData = [], isLoading: isAppsLoading } = useCanvases(organizationId);
  const { data: canvasFoldersData = [] } = useCanvasFolders(organizationId);

  const canvas = canvasData as PrototypeCanvas | undefined;
  const appFolderDotClasses = useMemo(() => {
    const folderColorById = new Map<string, CanvasFolderColor>();
    const folderColorByCanvasId = new Map<string, CanvasFolderColor>();

    for (const folder of canvasFoldersData) {
      const folderId = folder.metadata?.id;
      const folderColor = asCanvasFolderColor(folder.spec?.backgroundColor);

      if (folderId) {
        folderColorById.set(folderId, folderColor);
      }

      for (const folderCanvas of folder.spec?.canvases || []) {
        if (folderCanvas.id) {
          folderColorByCanvasId.set(folderCanvas.id, folderColor);
        }
      }
    }

    return new Map(
      (appsData as PrototypeCanvas[])
        .map((app): [string, string] | null => {
          const appId = app.metadata?.id;
          if (!appId) {
            return null;
          }

          const folderColor = folderColorByCanvasId.get(appId) || folderColorById.get(app.metadata?.folderId || "");

          return folderColor ? [appId, APP_SWITCHER_FOLDER_DOT_CLASSES[folderColor]] : null;
        })
        .filter((entry): entry is [string, string] => entry !== null),
    );
  }, [appsData, canvasFoldersData]);

  const apps = useMemo(() => {
    const recentIndex = new Map(recentAppIds.map((appId, index) => [appId, index]));

    return (appsData as PrototypeCanvas[])
      .filter((app) => app.metadata?.id && app.metadata?.name)
      .sort((left, right) => {
        const leftRecentIndex = recentIndex.get(left.metadata?.id || "") ?? Number.POSITIVE_INFINITY;
        const rightRecentIndex = recentIndex.get(right.metadata?.id || "") ?? Number.POSITIVE_INFINITY;

        if (leftRecentIndex !== rightRecentIndex) {
          return leftRecentIndex - rightRecentIndex;
        }

        return (left.metadata?.name || "").localeCompare(right.metadata?.name || "");
      });
  }, [appsData, recentAppIds]);

  const appName = canvas?.metadata?.name || (isCanvasLoading ? "Loading app..." : "Untitled app");

  useEffect(() => {
    setRecentAppIds(readRecentAppIds(organizationId));
  }, [organizationId]);

  useEffect(() => {
    if (!canvasId) {
      return;
    }

    setRecentAppIds(markAppRecentlyOpened(organizationId, canvasId));
  }, [organizationId, canvasId]);

  const agentState = useAgentState({
    isEditing: false,
    canvasVersion: canvas?.metadata?.liveVersionId || "",
    hideAddControls: false,
    readOnly: false,
    canvasId,
    organizationId,
  });

  const handleSelectApp = (nextCanvasId?: string) => {
    if (!nextCanvasId) {
      return;
    }

    setIsSwitcherOpen(false);
    navigate(`/${organizationId}/apps/${nextCanvasId}`);
  };

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-slate-100 text-slate-950">
      <AppTopBar
        organizationId={organizationId}
        appName={appName}
        isSwitcherOpen={isSwitcherOpen}
        onSwitcherOpenChange={setIsSwitcherOpen}
        apps={apps}
        appFolderDotClasses={appFolderDotClasses}
        isAppsLoading={isAppsLoading}
        activeCanvasId={canvasId}
        onSelectApp={handleSelectApp}
      />

      <AppBar activeTab={activeTab} onTabChange={setActiveTab} agentState={agentState} />

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <AgentSidebar
          agentState={agentState}
          showHeader={false}
          showCloseButton={false}
          showHeaderBorder={false}
          inputPlacement="bottom"
          compact
          sidebarWidthStorageKey="appPrototypeAgentSidebarWidth"
          defaultWidthPercent={0.4}
          showConversationList={false}
        />

        <main className="min-w-0 flex-1 overflow-auto">
          <AppTabContent activeTab={activeTab} />
        </main>
      </div>
    </div>
  );
}

function AppTopBar({
  organizationId,
  appName,
  isSwitcherOpen,
  onSwitcherOpenChange,
  apps,
  appFolderDotClasses,
  isAppsLoading,
  activeCanvasId,
  onSelectApp,
}: {
  organizationId: string;
  appName: string;
  isSwitcherOpen: boolean;
  onSwitcherOpenChange: (open: boolean) => void;
  apps: PrototypeCanvas[];
  appFolderDotClasses: Map<string, string>;
  isAppsLoading: boolean;
  activeCanvasId: string;
  onSelectApp: (canvasId?: string) => void;
}) {
  const switcherRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isSwitcherOpen) {
      return;
    }

    window.requestAnimationFrame(() => {
      switcherRef.current?.querySelector<HTMLInputElement>("[data-slot='command-input']")?.focus();
    });

    const handlePointerDown = (event: PointerEvent) => {
      if (!switcherRef.current?.contains(event.target as Node)) {
        onSwitcherOpenChange(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onSwitcherOpenChange(false);
      }
    };

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isSwitcherOpen, onSwitcherOpenChange]);

  return (
    <header className="relative z-30 flex h-9 shrink-0 items-center border-b border-slate-950/10 bg-white px-3">
      <div className="relative z-30 flex min-w-0 shrink-0 items-center">
        <OrganizationMenuButton organizationId={organizationId} className="[&_a_img]:h-6 [&_a_img]:w-6" />
      </div>

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-24">
        <div ref={switcherRef} className="pointer-events-auto relative">
          <button
            type="button"
            onClick={() => onSwitcherOpenChange(!isSwitcherOpen)}
            className="flex h-6 min-w-0 max-w-md items-center gap-2 rounded-md bg-transparent px-2.5 text-[13px] font-medium text-slate-800 outline outline-1 outline-slate-950/10 transition-colors hover:outline-slate-950/20"
            aria-label="Switch apps"
            aria-expanded={isSwitcherOpen}
          >
            <Search className="h-3 w-3 shrink-0 text-slate-500" />
            <span className="truncate">{appName}</span>
          </button>

          {isSwitcherOpen ? (
            <AppSwitcherMenu
              apps={apps}
              appFolderDotClasses={appFolderDotClasses}
              isLoading={isAppsLoading}
              activeCanvasId={activeCanvasId}
              onSelectApp={onSelectApp}
            />
          ) : null}
        </div>
      </div>
    </header>
  );
}

function AppBar({
  activeTab,
  onTabChange,
  agentState,
}: {
  activeTab: AppTab;
  onTabChange: (tab: AppTab) => void;
  agentState: ReturnType<typeof useAgentState>;
}) {
  const agentIsAvailable = agentState.showAgentSidebarToggle;
  const agentIsOpen = agentState.isAgentSidebarOpen;

  return (
    <div className="relative flex h-9 shrink-0 items-center border-b border-slate-950/10 bg-white px-3">
      <div className="relative z-10 flex shrink-0 items-center">
        <button
          type="button"
          disabled={!agentIsAvailable}
          aria-pressed={agentIsOpen}
          aria-label={agentIsOpen ? "Close Agent" : "Open Agent"}
          onClick={() => agentState.handleAgentSidebarOpenChange(!agentIsOpen)}
          className={cn(
            "-ml-1 flex h-6 w-6 items-center justify-center rounded-md text-slate-700 transition-colors hover:bg-slate-100 disabled:pointer-events-none disabled:opacity-50",
            agentIsOpen && "bg-violet-100 text-violet-600 hover:bg-violet-100",
          )}
        >
          <Sparkles className="h-3.5 w-3.5" />
        </button>
      </div>

      <nav className="pointer-events-none absolute inset-x-0 flex justify-center">
        <div className="pointer-events-auto flex items-center rounded-md bg-slate-100 p-0.5">
          {APP_TABS.map((tab) => {
            const selected = activeTab === tab.id;

            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => onTabChange(tab.id)}
                className={cn(
                  "flex h-6 items-center gap-1.5 rounded px-2.5 text-[13px] font-medium text-slate-600 transition-colors",
                  selected ? "bg-white text-slate-950 shadow-xs" : "hover:text-slate-950",
                )}
                aria-current={selected ? "page" : undefined}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
      </nav>
    </div>
  );
}

function AppTabContent({ activeTab }: { activeTab: AppTab }) {
  return <div className="min-h-full" aria-label={`${activeTab} tab content`} />;
}

function AppSwitcherMenu({
  apps,
  appFolderDotClasses,
  isLoading,
  activeCanvasId,
  onSelectApp,
}: {
  apps: PrototypeCanvas[];
  appFolderDotClasses: Map<string, string>;
  isLoading: boolean;
  activeCanvasId: string;
  onSelectApp: (canvasId?: string) => void;
}) {
  return (
    <Command className="absolute left-1/2 top-[-2px] z-50 h-[280px] w-[480px] -translate-x-1/2 rounded-md border border-slate-950/10 bg-white shadow-lg [&_[cmdk-group-heading]]:font-normal [&_[data-slot=command-input-wrapper]]:h-8 [&_[data-slot=command-input-wrapper]]:border-slate-950/10 [&_[data-slot=command-input-wrapper]]:px-2.5">
      <CommandInput placeholder="Search Apps..." className="h-8 py-1" />
      <CommandList className="max-h-none flex-1">
        <CommandEmpty>{isLoading ? "Loading apps..." : "No apps found."}</CommandEmpty>
        <CommandGroup heading="Recently opened">
          {apps.map((app) => {
            const appId = app.metadata?.id;
            const appName = app.metadata?.name || "Untitled app";
            const selected = appId === activeCanvasId;
            const folderDotClass = appId ? appFolderDotClasses.get(appId) : undefined;

            return (
              <CommandItem key={appId} value={appName} onSelect={() => onSelectApp(appId)}>
                <span
                  className={cn(
                    "h-3.5 w-3.5 shrink-0 rounded-full border border-slate-950/30 bg-white",
                    folderDotClass,
                  )}
                  aria-hidden="true"
                />
                <span className="truncate">{appName}</span>
                {selected ? <span className="ml-auto text-xs text-muted-foreground">Current</span> : null}
              </CommandItem>
            );
          })}
        </CommandGroup>
      </CommandList>
    </Command>
  );
}

function recentAppsStorageKey(organizationId: string) {
  return `${RECENT_APPS_STORAGE_KEY_PREFIX}:${organizationId}`;
}

function asCanvasFolderColor(value?: string): CanvasFolderColor {
  return CANVAS_FOLDER_COLORS.includes(value as CanvasFolderColor)
    ? (value as CanvasFolderColor)
    : DEFAULT_CANVAS_FOLDER_COLOR;
}

function readRecentAppIds(organizationId: string) {
  if (typeof window === "undefined") {
    return [];
  }

  try {
    const value = window.localStorage.getItem(recentAppsStorageKey(organizationId));
    const parsed = value ? JSON.parse(value) : [];

    return Array.isArray(parsed) ? parsed.filter((item): item is string => typeof item === "string") : [];
  } catch {
    return [];
  }
}

function markAppRecentlyOpened(organizationId: string, appId: string) {
  const recentAppIds = [
    appId,
    ...readRecentAppIds(organizationId).filter((recentAppId) => recentAppId !== appId),
  ].slice(0, MAX_RECENT_APPS);

  if (typeof window !== "undefined") {
    window.localStorage.setItem(recentAppsStorageKey(organizationId), JSON.stringify(recentAppIds));
  }

  return recentAppIds;
}
