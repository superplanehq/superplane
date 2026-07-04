import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Search, Settings2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSidebarLayoutStore, useSidebarLayoutViewport, useSidebarMount } from "@/stores/sidebarLayoutStore";
import { CategorySection } from "./CategorySection";
import { findFirstVisibleBlock, normalizeIntegrationName } from "./filter";
import type { BuildingBlock, BuildingBlockCategory } from "./types";
import { useSidebarSettings } from "./useSidebarSettings";

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
  canvasZoom: _canvasZoom = 1,
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
  disabled: boolean;
  disabledTooltip: string;
  onBlockClick?: (block: BuildingBlock) => void;
  onEnterSubmit?: (block: BuildingBlock) => void;
}

function OpenBuildingBlocksSidebar({
  onToggle,
  blocks,
  integrations,
  disabled,
  disabledTooltip,
  onBlockClick,
  onEnterSubmit,
}: OpenBuildingBlocksSidebarProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const sidebarRef = useRef<HTMLDivElement>(null);
  const activeResizePointerIdRef = useRef<number | null>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const {
    showIntegrationSetupStatus,
    setShowIntegrationSetupStatus,
    showConnectedIntegrationsOnTop,
    setShowConnectedIntegrationsOnTop,
  } = useSidebarSettings();

  const sidebarWidth = useSidebarLayoutStore((state) => state.rightWidth);
  const isResizing = useSidebarLayoutStore((state) => state.isRightResizing);
  const setRightResizing = useSidebarLayoutStore((state) => state.setRightResizing);
  const resizeRight = useSidebarLayoutStore((state) => state.resizeRight);

  useSidebarMount("right");
  useSidebarLayoutViewport();

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

  const updateSidebarWidthFromPointer = useCallback(
    (clientX: number) => {
      resizeRight(window.innerWidth - clientX);
    },
    [resizeRight],
  );

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
      setRightResizing(false);
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
  }, [isResizing, updateSidebarWidthFromPointer, setRightResizing]);

  const handlePointerDown = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      event.preventDefault();
      activeResizePointerIdRef.current = event.pointerId;
      updateSidebarWidthFromPointer(event.clientX);
      setRightResizing(true);
    },
    [updateSidebarWidthFromPointer, setRightResizing],
  );

  const sortedCategories = useMemo(() => {
    const categoryOrder: Record<string, number> = {
      Core: 0,
      Runners: 1,
      Debugging: 2,
      Memory: 3,
    };

    const isConnectedCategory = (category: BuildingBlockCategory) => {
      const integrationName = category.blocks.find((block) => block.integrationName)?.integrationName;
      if (!integrationName) {
        return false;
      }
      return integrations.some(
        (integration) =>
          normalizeIntegrationName(integration.metadata?.integrationName) === normalizeIntegrationName(integrationName),
      );
    };

    return [...blocks].sort((a, b) => {
      const aOrder = categoryOrder[a.name] ?? Infinity;
      const bOrder = categoryOrder[b.name] ?? Infinity;

      if (aOrder !== bOrder) {
        return aOrder - bOrder;
      }

      if (showConnectedIntegrationsOnTop && aOrder === Infinity && bOrder === Infinity) {
        const aConnected = isConnectedCategory(a);
        const bConnected = isConnectedCategory(b);
        if (aConnected !== bConnected) {
          return aConnected ? -1 : 1;
        }
      }

      return a.name.localeCompare(b.name);
    });
  }, [blocks, integrations, showConnectedIntegrationsOnTop]);

  return (
    <div
      ref={sidebarRef}
      className="absolute right-0 top-0 h-full z-21"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="building-blocks-sidebar"
    >
      <div
        onPointerDown={handlePointerDown}
        className="group absolute left-0 top-0 bottom-0 z-40 w-4 cursor-col-resize touch-none bg-transparent"
        style={{ marginLeft: "-8px" }}
      >
        <div
          aria-hidden
          className={`pointer-events-none absolute top-0 bottom-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-slate-950/50 ${
            isResizing ? "bg-slate-950/50" : ""
          }`}
        />
      </div>

      <div className="border-l-1 border-border h-full flex flex-col overflow-hidden bg-white">
        <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
          <h2 className="min-w-0 text-sm font-medium">Select Component</h2>
          <button
            type="button"
            onClick={() => onToggle(false)}
            data-testid="close-sidebar-button"
            className="shrink-0 flex h-6 w-6 cursor-pointer items-center justify-center rounded leading-none hover:bg-slate-950/5"
            aria-label="Close sidebar"
          >
            <X size={16} className="shrink-0" />
          </button>
        </div>

        <div className="flex flex-1 flex-col min-h-0 overflow-y-auto overflow-x-hidden">
          <div className="flex items-center gap-2 px-5 pt-3 shrink-0">
            <div className="flex-1 relative min-w-0">
              <Search
                className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none"
                size={16}
              />
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
                  const firstBlock = findFirstVisibleBlock(sortedCategories, searchTerm, "all");
                  if (!firstBlock) {
                    return;
                  }
                  e.preventDefault();
                  onEnterSubmit(firstBlock);
                }}
              />
            </div>
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
      </div>
    </div>
  );
}
