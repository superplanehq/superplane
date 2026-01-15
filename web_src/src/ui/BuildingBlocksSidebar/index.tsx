import type { SuperplaneBlueprintsOutputChannel, SuperplaneComponentsOutputChannel } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { resolveIcon } from "@/lib/utils";
import { isCustomComponentsEnabled } from "@/lib/env";
import { getBackgroundColorClass } from "@/utils/colors";
import { ChevronRight, GripVerticalIcon, Plus, Search, StickyNote, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toTestId } from "../../utils/testID";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { ComponentBase } from "../componentBase";
import githubIcon from "@/assets/icons/integrations/github.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";

export interface BuildingBlock {
  name: string;
  label?: string;
  description?: string;
  type: "trigger" | "component" | "blueprint";
  outputChannels?: Array<SuperplaneComponentsOutputChannel | SuperplaneBlueprintsOutputChannel>;
  configuration?: any[];
  icon?: string;
  color?: string;
  id?: string; // for blueprints
  isLive?: boolean; // marks items that actually work now
  appName?: string; // for components/triggers from applications
  deprecated?: boolean; // marks items that are deprecated
}

export type BuildingBlockCategory = {
  name: string;
  blocks: BuildingBlock[];
};

export interface BuildingBlocksSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  canvasZoom?: number;
  disabled?: boolean;
  onBlockClick?: (block: BuildingBlock) => void;
  onAddNote?: () => void;
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  canvasZoom = 1,
  disabled = false,
  onBlockClick,
  onAddNote,
}: BuildingBlocksSidebarProps) {
  if (!isOpen) {
    return (
      <div className="absolute top-4 right-4 z-10 flex gap-3">
        <Button variant="outline" onClick={onAddNote} aria-label="Add Note" data-testid="add-note-button">
          <StickyNote size={16} className="animate-pulse" />
          Add Note
        </Button>
        <Button
          variant="outline"
          onClick={() => onToggle(true)}
          aria-label="Open sidebar"
          data-testid="open-sidebar-button"
        >
          <Plus size={16} />
          Components
        </Button>
      </div>
    );
  }

  const [searchTerm, setSearchTerm] = useState("");
  const sidebarRef = useRef<HTMLDivElement>(null);
  const isDraggingRef = useRef(false);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 450;
  });
  const [isResizing, setIsResizing] = useState(false);
  const [hoveredBlock, setHoveredBlock] = useState<BuildingBlock | null>(null);
  const dragPreviewRef = useRef<HTMLDivElement>(null);

  // Save sidebar width to localStorage whenever it changes
  useEffect(() => {
    localStorage.setItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  // Handle resize mouse events
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;

      const newWidth = window.innerWidth - e.clientX;
      // Set min width to 320px and max width to 600px
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

  const categoryOrder: Record<string, number> = {
    Primitives: 0,
    Triggers: 1,
    Bundles: 2,
  };

  const filteredCategories = (blocks || []).filter((category) => {
    if (category.name === "Bundles" && !isCustomComponentsEnabled()) {
      return false;
    }
    return true;
  });

  const sortedCategories = [...filteredCategories].sort((a, b) => {
    const aOrder = categoryOrder[a.name] ?? Infinity;
    const bOrder = categoryOrder[b.name] ?? Infinity;

    if (aOrder !== bOrder) {
      return aOrder - bOrder;
    }

    return a.name.localeCompare(b.name);
  });

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-20 overflow-y-auto overflow-x-hidden bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="building-blocks-sidebar"
    >
      {/* Resize handle */}
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

      {/* Header */}
      <div className="flex items-center justify-between gap-3 px-5 py-4 relative">
        <div className="flex flex-col items-start gap-3 w-full">
          <div className="flex justify-between gap-3 w-full">
            <div className="flex flex-col gap-0.5">
              <h2 className="text-base font-semibold">New Component</h2>
            </div>
          </div>
          <div
            onClick={() => onToggle(false)}
            className="absolute top-4 right-4 w-6 h-6 hover:bg-slate-950/5 rounded flex items-center justify-center cursor-pointer leading-none"
          >
            <X size={16} />
          </div>
        </div>
      </div>

      {/* Search */}
      <div className="flex items-center gap-2 px-5">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={16} />
          <input
            type="text"
            placeholder="Filter components..."
            className="w-full pl-9 pr-3 py-2 text-sm border border-gray-300 dark:border-gray-700 rounded-md bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-100 placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-gray-800 focus:border-transparent"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
      </div>

      <div className="gap-2 py-6">
        {sortedCategories.map((category) => (
          <CategorySection
            key={category.name}
            category={category}
            canvasZoom={canvasZoom}
            searchTerm={searchTerm}
            isDraggingRef={isDraggingRef}
            setHoveredBlock={setHoveredBlock}
            dragPreviewRef={dragPreviewRef}
            onBlockClick={onBlockClick}
          />
        ))}

        {/* Disabled overlay - only over items */}
        {disabled && (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="absolute inset-0 bg-white/60 dark:bg-gray-900/60 z-30 cursor-not-allowed" />
            </TooltipTrigger>
            <TooltipContent side="left" sideOffset={10}>
              <p>Finish configuring the selected component first</p>
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      {/* Hidden drag preview - pre-rendered and ready for drag */}
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
            headerColor="bg-white"
            iconSlug={hoveredBlock.icon}
            iconColor="text-gray-800"
            collapsedBackground={getBackgroundColorClass("white")}
            hideActionsButton={true}
            includeEmptyState={true}
            collapsed={false}
          />
        )}
      </div>
    </div>
  );
}

interface CategorySectionProps {
  category: BuildingBlockCategory;
  canvasZoom: number;
  searchTerm?: string;
  isDraggingRef: React.RefObject<boolean>;
  setHoveredBlock: (block: BuildingBlock | null) => void;
  dragPreviewRef: React.RefObject<HTMLDivElement | null>;
  onBlockClick?: (block: BuildingBlock) => void;
}

function CategorySection({
  category,
  canvasZoom,
  searchTerm = "",
  isDraggingRef,
  setHoveredBlock,
  dragPreviewRef,
  onBlockClick,
}: CategorySectionProps) {
  const query = searchTerm.trim().toLowerCase();
  const categoryMatches = query ? (category.name || "").toLowerCase().includes(query) : true;

  const baseBlocks = categoryMatches
    ? category.blocks || []
    : (category.blocks || []).filter((block) => {
        const name = (block.name || "").toLowerCase();
        const label = (block.label || "").toLowerCase();
        return name.includes(query) || label.includes(query);
      });

  // Only show live/ready blocks
  const allBlocks = baseBlocks.filter((b) => b.isLive);

  if (allBlocks.length === 0) {
    return null;
  }

  return (
    <details className="flex-1 px-5 mb-5" open>
      <summary className="relative cursor-pointer hover:text-gray-500 dark:hover:text-gray-300 mb-3 flex items-center gap-1 [&::-webkit-details-marker]:hidden [&::marker]:hidden before:absolute before:inset-x-0 before:top-1/2 before:-translate-y-1/2 before:h-px before:bg-slate-300 dark:before:bg-gray-800 before:content-[''] before:transform">
        <span className="relative z-10 flex items-center gap-1 bg-white dark:bg-gray-900 pr-3">
          <ChevronRight className="h-3 w-3 transition-transform [[details[open]]>&]:rotate-90" />
          <span className="text-[13px] text-gray-500 font-medium pl-1">{category.name}</span>
        </span>
      </summary>

      <ItemGroup>
        {allBlocks.map((block) => {
          const iconSlug = block.type === "blueprint" ? "component" : block.icon || "zap";
          
          // Use SVG icons for application components/triggers
          const appLogoMap: Record<string, string> = {
            github: githubIcon,
            semaphore: SemaphoreLogo,
            pagerduty: pagerDutyIcon,
            dash0: dash0Icon,
          };
          const appIconSrc = block.appName ? appLogoMap[block.appName] : undefined;
          const IconComponent = resolveIcon(iconSlug);

          const isLive = !!block.isLive;
          return (
            <Item
              data-testid={toTestId(`building-block-${block.name}`)}
              key={`${block.type}-${block.name}`}
              draggable={isLive}
              onClick={() => {
                if (isLive && onBlockClick) {
                  onBlockClick(block);
                }
              }}
              onMouseEnter={() => {
                if (isLive) {
                  setHoveredBlock(block);
                }
              }}
              onMouseLeave={() => {
                setHoveredBlock(null);
              }}
              onDragStart={(e) => {
                if (!isLive) {
                  e.preventDefault();
                  return;
                }
                isDraggingRef.current = true;
                e.dataTransfer.effectAllowed = "move";
                e.dataTransfer.setData("application/reactflow", JSON.stringify(block));

                // Use the pre-rendered drag preview
                const previewElement = dragPreviewRef.current?.firstChild as HTMLElement;
                if (previewElement) {
                  // Clone the pre-rendered element
                  const clone = previewElement.cloneNode(true) as HTMLElement;

                  // Create a container div to hold the scaled element
                  const container = document.createElement("div");
                  container.style.cssText = `
                    position: absolute;
                    top: -10000px;
                    left: -10000px;
                    pointer-events: none;
                  `;

                  // Apply zoom and opacity to the clone
                  clone.style.transform = `scale(${canvasZoom})`;
                  clone.style.transformOrigin = "top left";
                  clone.style.opacity = "0.85";

                  container.appendChild(clone);
                  document.body.appendChild(container);

                  // Get dimensions for centering
                  const rect = previewElement.getBoundingClientRect();
                  const offsetX = (rect.width / 2) * canvasZoom;
                  const offsetY = 30 * canvasZoom;
                  e.dataTransfer.setDragImage(container, offsetX, offsetY);

                  // Cleanup after drag starts
                  setTimeout(() => {
                    if (document.body.contains(container)) {
                      document.body.removeChild(container);
                    }
                  }, 0);
                }
              }}
              onDragEnd={() => {
                isDraggingRef.current = false;
                setHoveredBlock(null);
              }}
              aria-disabled={!isLive}
              title={isLive ? undefined : "Coming soon"}
              className={`ml-3 px-2 py-1 flex items-center gap-2 cursor-grab active:cursor-grabbing hover:bg-sky-100`}
              size="sm"
            >
              <ItemMedia>
                {appIconSrc ? (
                  <img src={appIconSrc} alt={block.label || block.name} className="w-3.5 h-3.5" />
                ) : (
                  <IconComponent size={14} className="text-gray-500" />
                )}
              </ItemMedia>

              <ItemContent>
                <div className="flex items-center gap-2">
                  <ItemTitle className="text-sm font-normal">{block.label || block.name}</ItemTitle>
                  {block.deprecated && (
                    <span className="px-1.5 py-0.5 text-[11px] font-medium bg-gray-950/5 text-gray-500 rounded whitespace-nowrap">
                      Deprecated
                    </span>
                  )}
                </div>
              </ItemContent>

              <GripVerticalIcon className="text-gray-500 hover:text-gray-800" size={14} />
            </Item>
          );
        })}
      </ItemGroup>
    </details>
  );
}
