import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getBackgroundColorClass } from "@/lib/colors";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Search, Settings2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { ComponentBase } from "../componentBase";
import { CategorySection } from "./CategorySection";
import { findFirstVisibleBlock, type TypeFilter } from "./filter";
import type { BuildingBlock, BuildingBlockCategory } from "./types";

export type { AgentContext, AgentMode } from "@/components/AgentSidebar/agentChat";
export type { BuildingBlock, BuildingBlockCategory } from "./types";

export interface BuildingBlocksSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  integrations?: OrganizationsIntegration[];
  canvasZoom?: number;
  disabled?: boolean;
  disabledMessage?: string;
  onBlockClick?: (block: BuildingBlock) => void;
  /**
   * Called when the user submits the filter input (presses Enter) and at least
   * one block matches the current filter. Receives the first visible block in
   * the same order the sidebar renders them. No-op when the filter is empty
   * or has zero matches — the caller never has to handle a "no block" case.
   */
  onEnterSubmit?: (block: BuildingBlock) => void;
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  integrations = [],
  canvasZoom = 1,
  disabled = false,
  disabledMessage,
  onBlockClick,
  onEnterSubmit,
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
      canvasZoom={canvasZoom}
      disabled={disabled}
      disabledTooltip={disabledTooltip}
      onBlockClick={onBlockClick}
      onEnterSubmit={onEnterSubmit}
    />
  );
}

interface OpenBuildingBlocksSidebarProps {
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  integrations: OrganizationsIntegration[];
  canvasZoom: number;
  disabled: boolean;
  disabledTooltip: string;
  onBlockClick?: (block: BuildingBlock) => void;
  onEnterSubmit?: (block: BuildingBlock) => void;
}

function OpenBuildingBlocksSidebar({
  onToggle,
  blocks,
  integrations,
  canvasZoom,
  disabled,
  disabledTooltip,
  onBlockClick,
  onEnterSubmit,
}: OpenBuildingBlocksSidebarProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const [typeFilter, setTypeFilter] = useState<TypeFilter>("all");
  const sidebarRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const isDraggingRef = useRef(false);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") {
      return 450;
    }

    const saved = window.localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 450;
  });
  const [isResizing, setIsResizing] = useState(false);
  const [hoveredBlock, setHoveredBlock] = useState<BuildingBlock | null>(null);
  const dragPreviewRef = useRef<HTMLDivElement>(null);
  const [showIntegrationSetupStatus, setShowIntegrationSetupStatus] = useState(true);
  const [showConnectedIntegrationsOnTop, setShowConnectedIntegrationsOnTop] = useState(false);

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

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;

      const newWidth = window.innerWidth - e.clientX;
      const clampedWidth = Math.max(320, Math.min(600, newWidth));
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";

      return () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
    }
  }, [isResizing, handleMouseMove, handleMouseUp]);

  const normalizeIntegrationName = (value?: string) => (value || "").toLowerCase().replace(/[^a-z0-9]/g, "");

  const sortedCategories = useMemo(() => {
    const categoryOrder: Record<string, number> = {
      Core: 0,
      Memory: 1,
    };

    return [...blocks].sort((a, b) => {
      const aOrder = categoryOrder[a.name] ?? Infinity;
      const bOrder = categoryOrder[b.name] ?? Infinity;

      if (aOrder !== bOrder) {
        return aOrder - bOrder;
      }

      if (showConnectedIntegrationsOnTop && aOrder === Infinity && bOrder === Infinity) {
        const aIntegrationName = a.blocks.find((block) => block.integrationName)?.integrationName;
        const bIntegrationName = b.blocks.find((block) => block.integrationName)?.integrationName;

        const aHasConnectedIntegration = aIntegrationName
          ? integrations.some(
              (integration) =>
                normalizeIntegrationName(integration.spec?.integrationName) ===
                normalizeIntegrationName(aIntegrationName),
            )
          : false;

        const bHasConnectedIntegration = bIntegrationName
          ? integrations.some(
              (integration) =>
                normalizeIntegrationName(integration.spec?.integrationName) ===
                normalizeIntegrationName(bIntegrationName),
            )
          : false;

        if (aHasConnectedIntegration !== bHasConnectedIntegration) {
          return aHasConnectedIntegration ? -1 : 1;
        }
      }

      return a.name.localeCompare(b.name);
    });
  }, [blocks, integrations, showConnectedIntegrationsOnTop]);

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="building-blocks-sidebar"
    >
      <div
        onMouseDown={handleMouseDown}
        className={`absolute left-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-gray-100 transition-colors flex items-center justify-center group ${
          isResizing ? "bg-blue-50" : ""
        }`}
        style={{ marginLeft: "-8px" }}
      >
        <div
          className={`w-2 h-14 rounded-full bg-gray-300 group-hover:bg-gray-800 transition-colors ${
            isResizing ? "bg-blue-500" : ""
          }`}
        />
      </div>

      <div className="flex items-center justify-between gap-3 px-5 py-4 shrink-0 border-b border-border/60">
        <div className="flex flex-col items-start gap-0.5 min-w-0">
          <h2 className="text-base font-medium">Add Component</h2>
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
              onKeyDown={(e) => {
                if (e.key !== "Enter" || disabled || !onEnterSubmit) {
                  return;
                }
                if (searchTerm.trim().length === 0) {
                  return;
                }
                const firstBlock = findFirstVisibleBlock(sortedCategories, searchTerm, typeFilter);
                if (!firstBlock) {
                  return;
                }
                e.preventDefault();
                onEnterSubmit(firstBlock);
              }}
            />
          </div>
          <Select value={typeFilter} onValueChange={(value) => setTypeFilter(value as typeof typeFilter)}>
            <SelectTrigger size="sm" className="w-[120px] shrink-0">
              <SelectValue placeholder="All Types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Types</SelectItem>
              <SelectItem value="trigger">Trigger</SelectItem>
              <SelectItem value="component">Action</SelectItem>
            </SelectContent>
          </Select>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon-sm" className="h-8 w-8 shrink-0" aria-label="Sidebar settings">
                <Settings2 className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuCheckboxItem
                checked={showIntegrationSetupStatus}
                onCheckedChange={(checked) => setShowIntegrationSetupStatus(Boolean(checked))}
              >
                Show integration setup status
              </DropdownMenuCheckboxItem>
              <DropdownMenuCheckboxItem
                checked={showConnectedIntegrationsOnTop}
                onCheckedChange={(checked) => setShowConnectedIntegrationsOnTop(Boolean(checked))}
              >
                Connected integrations on top
              </DropdownMenuCheckboxItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="relative flex-1 min-h-0 gap-2 py-6">
          {sortedCategories.map((category) => (
            <CategorySection
              key={category.name}
              category={category}
              integrations={integrations}
              showIntegrationSetupStatus={showIntegrationSetupStatus}
              canvasZoom={canvasZoom}
              searchTerm={searchTerm}
              typeFilter={typeFilter}
              isDraggingRef={isDraggingRef}
              setHoveredBlock={setHoveredBlock}
              dragPreviewRef={dragPreviewRef}
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

      <div
        ref={dragPreviewRef}
        style={{
          position: "absolute",
          top: "-10000px",
          left: "-10000px",
          pointerEvents: "none",
        }}
      >
        {hoveredBlock && (
          <ComponentBase
            title={hoveredBlock.label || hoveredBlock.name || "New Component"}
            iconSlug={hoveredBlock.name?.split(".")[0] === "smtp" ? "mail" : (hoveredBlock.icon ?? "zap")}
            iconColor="text-gray-800"
            collapsedBackground={getBackgroundColorClass("white")}
            includeEmptyState={true}
            collapsed={false}
          />
        )}
      </div>
    </div>
  );
}
