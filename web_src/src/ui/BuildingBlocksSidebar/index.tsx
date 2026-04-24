import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { resolveIcon } from "@/lib/utils";
import { Search, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { CategorySection } from "./CategorySection";
import { getIntegrationIconSrc } from "../componentSidebar/integrationIcons";
import type { BuildingBlock, BuildingBlockCategory } from "./types";

const DEFAULT_COMPONENT_SIDEBAR_WIDTH = 380;
const MIN_COMPONENT_SIDEBAR_WIDTH = 300;
const MAX_COMPONENT_SIDEBAR_WIDTH = 800;

export type { AgentContext, AgentMode } from "@/components/AgentSidebar/agentChat";
export type { BuildingBlock, BuildingBlockCategory } from "./types";

export interface BuildingBlocksSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  integrations?: OrganizationsIntegration[];
  availableIntegrations?: IntegrationsIntegrationDefinition[];
  integrationDialogOpen?: boolean;
  disabled?: boolean;
  disabledMessage?: string;
  onBlockClick?: (block: BuildingBlock) => void;
  onConnectIntegration?: (integrationName: string) => void;
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  integrations = [],
  availableIntegrations = [],
  integrationDialogOpen = false,
  disabled = false,
  disabledMessage,
  onBlockClick,
  onConnectIntegration,
}: BuildingBlocksSidebarProps) {
  const disabledTooltip = disabledMessage || "Finish configuring the selected component first";

  if (!isOpen) {
    return null;
  }

  return (
    <OpenBuildingBlocksSidebar
      onToggle={onToggle}
      blocks={blocks}
      integrations={integrations}
      availableIntegrations={availableIntegrations}
      integrationDialogOpen={integrationDialogOpen}
      disabled={disabled}
      disabledTooltip={disabledTooltip}
      onBlockClick={onBlockClick}
      onConnectIntegration={onConnectIntegration}
    />
  );
}

interface OpenBuildingBlocksSidebarProps {
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  integrations: OrganizationsIntegration[];
  availableIntegrations: IntegrationsIntegrationDefinition[];
  integrationDialogOpen: boolean;
  disabled: boolean;
  disabledTooltip: string;
  onBlockClick?: (block: BuildingBlock) => void;
  onConnectIntegration?: (integrationName: string) => void;
}

function OpenBuildingBlocksSidebar({
  onToggle,
  blocks,
  integrations,
  availableIntegrations,
  integrationDialogOpen,
  disabled,
  disabledTooltip,
  onBlockClick,
  onConnectIntegration,
}: OpenBuildingBlocksSidebarProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const sidebarRef = useRef<HTMLDivElement>(null);
  const activeResizePointerIdRef = useRef<number | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") {
      return DEFAULT_COMPONENT_SIDEBAR_WIDTH;
    }

    const saved = window.localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    const parsed = saved ? Number.parseInt(saved, 10) : NaN;
    if (!Number.isFinite(parsed)) {
      return DEFAULT_COMPONENT_SIDEBAR_WIDTH;
    }
    return Math.max(MIN_COMPONENT_SIDEBAR_WIDTH, Math.min(MAX_COMPONENT_SIDEBAR_WIDTH, parsed));
  });
  const [isResizing, setIsResizing] = useState(false);
  const [isIntegrationsModalOpen, setIsIntegrationsModalOpen] = useState(false);
  const shouldReopenModalAfterIntegrationDialogRef = useRef(false);
  const previousIntegrationDialogOpenRef = useRef(integrationDialogOpen);

  useEffect(() => {
    localStorage.setItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (!searchInputRef.current) {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      searchInputRef.current?.focus();
    }, 100);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, []);

  const updateSidebarWidthFromPointer = useCallback((clientX: number) => {
    const newWidth = window.innerWidth - clientX;
    const clampedWidth = Math.max(MIN_COMPONENT_SIDEBAR_WIDTH, Math.min(MAX_COMPONENT_SIDEBAR_WIDTH, newWidth));
    setSidebarWidth(clampedWidth);
  }, []);

  useEffect(() => {
    if (!isResizing) {
      return;
    }

    const handleWindowPointerMove = (event: PointerEvent) => {
      if (activeResizePointerIdRef.current !== null && event.pointerId !== activeResizePointerIdRef.current) {
        return;
      }
      updateSidebarWidthFromPointer(event.clientX);
    };

    const finishResize = (event: PointerEvent) => {
      if (activeResizePointerIdRef.current !== null && event.pointerId !== activeResizePointerIdRef.current) {
        return;
      }
      activeResizePointerIdRef.current = null;
      setIsResizing(false);
    };

    window.addEventListener("pointermove", handleWindowPointerMove);
    window.addEventListener("pointerup", finishResize);
    window.addEventListener("pointercancel", finishResize);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";

    return () => {
      window.removeEventListener("pointermove", handleWindowPointerMove);
      window.removeEventListener("pointerup", finishResize);
      window.removeEventListener("pointercancel", finishResize);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, updateSidebarWidthFromPointer]);

  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      e.preventDefault();
      activeResizePointerIdRef.current = e.pointerId;
      updateSidebarWidthFromPointer(e.clientX);
      setIsResizing(true);
    },
    [updateSidebarWidthFromPointer],
  );

  const normalizeIntegrationName = (value?: string) => (value || "").toLowerCase().replace(/[^a-z0-9]/g, "");

  const connectedIntegrationNames = useMemo(() => {
    const connected = new Set<string>();
    integrations.forEach((integration) => {
      if (integration.status?.state === "ready") {
        connected.add(normalizeIntegrationName(integration.spec?.integrationName));
      }
    });
    return connected;
  }, [integrations]);

  const integrationConnectionStateByName = useMemo(() => {
    const map = new Map<string, "ready" | "pending" | "error">();
    integrations.forEach((integration) => {
      const normalizedName = normalizeIntegrationName(integration.spec?.integrationName);
      if (!normalizedName) {
        return;
      }

      const state = integration.status?.state;
      const nextState = state === "ready" ? "ready" : state === "error" ? "error" : "pending";
      const current = map.get(normalizedName);

      if (!current) {
        map.set(normalizedName, nextState);
        return;
      }

      if (current === "ready") {
        return;
      }
      if (current === "pending" && nextState === "error") {
        return;
      }
      map.set(normalizedName, nextState);
    });
    return map;
  }, [integrations]);

  const configuredIntegrationNames = useMemo(() => {
    const configured = new Set<string>();
    integrations.forEach((integration) => {
      configured.add(normalizeIntegrationName(integration.spec?.integrationName));
    });
    return configured;
  }, [integrations]);

  const sortedCategories = useMemo(() => {
    const categoryOrder: Record<string, number> = {
      Core: 0,
      Memory: 1,
    };

    const visibleCategories = blocks.filter((category) => {
      if (category.name === "Core" || category.name === "Memory") {
        return true;
      }

      return category.blocks.some((block) =>
        configuredIntegrationNames.has(normalizeIntegrationName(block.integrationName)),
      );
    });

    return [...visibleCategories].sort((a, b) => {
      const aOrder = categoryOrder[a.name] ?? Infinity;
      const bOrder = categoryOrder[b.name] ?? Infinity;

      if (aOrder !== bOrder) {
        return aOrder - bOrder;
      }

      return a.name.localeCompare(b.name);
    });
  }, [blocks, configuredIntegrationNames]);

  const integrationsModalItems = useMemo(() => {
    return [...availableIntegrations].sort((a, b) => (a.label || a.name || "").localeCompare(b.label || b.name || ""));
  }, [availableIntegrations]);
  const connectedIntegrationItems = useMemo(() => {
    return integrationsModalItems.filter((integration) =>
      integrationConnectionStateByName.has(normalizeIntegrationName(integration.name)),
    );
  }, [integrationsModalItems, integrationConnectionStateByName]);

  useEffect(() => {
    const previous = previousIntegrationDialogOpenRef.current;
    if (previous && !integrationDialogOpen && shouldReopenModalAfterIntegrationDialogRef.current) {
      setIsIntegrationsModalOpen(true);
      shouldReopenModalAfterIntegrationDialogRef.current = false;
    }
    previousIntegrationDialogOpenRef.current = integrationDialogOpen;
  }, [integrationDialogOpen]);

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="building-blocks-sidebar"
    >
      <div
        onPointerDown={handlePointerDown}
        className={`absolute left-0 top-0 bottom-0 w-5 cursor-ew-resize touch-none hover:bg-gray-100 transition-colors flex items-center justify-center group z-40 ${
          isResizing ? "bg-blue-50" : ""
        }`}
        style={{ marginLeft: "-10px" }}
      >
        <div
          className={`w-2 h-14 rounded-full bg-gray-300 group-hover:bg-gray-800 transition-colors ${
            isResizing ? "bg-blue-500" : ""
          }`}
        />
      </div>

      <div className="flex items-center justify-between gap-3 px-5 py-2.5 shrink-0">
        <div className="flex flex-col items-start gap-0.5 min-w-0">
          <h2 className="text-sm font-medium">Select Component</h2>
        </div>
        <button
          type="button"
          onClick={() => onToggle(false)}
          data-testid="close-sidebar-button"
          className="shrink-0 z-40 w-8 h-8 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer leading-none border border-transparent text-muted-foreground"
          aria-label="Close sidebar"
        >
          <X size={16} />
        </button>
      </div>

      <div className="flex flex-1 flex-col min-h-0 overflow-y-auto overflow-x-hidden">
        <div className="flex items-center gap-2 px-5 pt-3 shrink-0">
          <div className="flex-1 relative min-w-0">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" size={16} />
            <Input
              ref={searchInputRef}
              type="text"
              placeholder="Filter components..."
              className="pl-9"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-8 shrink-0 border border-transparent bg-white px-3 text-slate-700 [background:linear-gradient(white,white)_padding-box,linear-gradient(90deg,#f97316,#eab308,#22c55e,#06b6d4,#3b82f6,#a855f7,#ec4899)_border-box]"
            onClick={() => setIsIntegrationsModalOpen(true)}
          >
            + Integrations
          </Button>
        </div>

        <div className="relative flex-1 min-h-0 gap-2 py-6">
          {sortedCategories.map((category) => (
            <CategorySection
              key={category.name}
              category={category}
              searchTerm={searchTerm}
              onBlockClick={onBlockClick}
            />
          ))}

          {disabled && (
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="absolute inset-0 bg-white/60 dark:bg-gray-900/60 z-30 cursor-not-allowed" />
              </TooltipTrigger>
              <TooltipContent side="left" sideOffset={10}>
                <p>{disabledTooltip}</p>
              </TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>

      <Dialog open={isIntegrationsModalOpen} onOpenChange={setIsIntegrationsModalOpen}>
        <DialogContent className="w-full max-w-[calc(100%-2rem)] pb-0 sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle className="text-base font-medium">Connect Integrations</DialogTitle>
          </DialogHeader>
          <div className="max-h-[68vh] overflow-y-auto space-y-6 pr-1">
            {connectedIntegrationItems.length > 0 ? (
              <IntegrationGridSection
                title="Connected"
                items={connectedIntegrationItems}
                connectedIntegrationNames={connectedIntegrationNames}
                integrationConnectionStateByName={integrationConnectionStateByName}
                normalizeIntegrationName={normalizeIntegrationName}
                onSelect={(integrationName) => {
                  if (!onConnectIntegration) {
                    return;
                  }
                  shouldReopenModalAfterIntegrationDialogRef.current = true;
                  onConnectIntegration(integrationName);
                  setIsIntegrationsModalOpen(false);
                }}
              />
            ) : null}

            <IntegrationGridSection
              title="All Integrations"
              items={integrationsModalItems}
              connectedIntegrationNames={connectedIntegrationNames}
              integrationConnectionStateByName={integrationConnectionStateByName}
              normalizeIntegrationName={normalizeIntegrationName}
              onSelect={(integrationName) => {
                if (!onConnectIntegration) {
                  return;
                }
                shouldReopenModalAfterIntegrationDialogRef.current = true;
                onConnectIntegration(integrationName);
                setIsIntegrationsModalOpen(false);
              }}
            />

            {integrationsModalItems.length === 0 ? (
              <div className="rounded-md border border-dashed border-slate-300 px-3 py-4 text-sm text-slate-500">
                No integrations available.
              </div>
            ) : null}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function IntegrationGridSection({
  title,
  items,
  connectedIntegrationNames,
  integrationConnectionStateByName,
  normalizeIntegrationName,
  onSelect,
}: {
  title: string;
  items: IntegrationsIntegrationDefinition[];
  connectedIntegrationNames: Set<string>;
  integrationConnectionStateByName: Map<string, "ready" | "pending" | "error">;
  normalizeIntegrationName: (value?: string) => string;
  onSelect: (integrationName: string) => void;
}) {
  const FallbackIcon = resolveIcon("puzzle");

  return (
    <section className="space-y-3">
      <h3 className="text-sm font-medium text-slate-500">{title}</h3>
      <div className="grid grid-cols-2 gap-3 pb-6 sm:grid-cols-3 lg:grid-cols-4">
        {items.map((integration) => {
          const normalizedName = normalizeIntegrationName(integration.name);
          const isConnected = connectedIntegrationNames.has(normalizedName);
          const connectionState = integrationConnectionStateByName.get(normalizedName);
          const iconSrc = getIntegrationIconSrc(integration.name);

          return (
            <button
              key={`${title}-${integration.name}`}
              type="button"
              onClick={() => {
                if (!integration.name) {
                  return;
                }
                onSelect(integration.name);
              }}
              className="relative flex h-28 flex-col items-center justify-center gap-2 rounded-md border border-slate-200 px-3 py-2 text-center transition-colors hover:bg-slate-50"
            >
              {iconSrc ? (
                <img src={iconSrc} alt={integration.label || integration.name} className="size-7" />
              ) : (
                <FallbackIcon className="size-7 text-slate-500" />
              )}
              <span className="line-clamp-2 text-xs font-medium text-slate-700">
                {integration.label || integration.name}
              </span>
              {connectionState ? (
                <span
                  className={`absolute right-2 top-2 rounded-full px-1.5 py-0.5 text-[10px] font-semibold ${
                    connectionState === "ready"
                      ? "bg-green-100 text-green-700"
                      : connectionState === "pending"
                        ? "bg-amber-100 text-amber-700"
                        : "bg-red-100 text-red-700"
                  }`}
                >
                  {connectionState === "ready" ? "Connected" : connectionState === "pending" ? "Pending" : "Error"}
                </span>
              ) : isConnected ? (
                <span className="absolute right-2 top-2 rounded-full bg-green-100 px-1.5 py-0.5 text-[10px] font-semibold text-green-700">
                  Connected
                </span>
              ) : null}
            </button>
          );
        })}
      </div>
    </section>
  );
}
