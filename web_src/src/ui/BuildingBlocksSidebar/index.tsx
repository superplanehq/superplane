import type { SuperplaneBlueprintsOutputChannel, SuperplaneComponentsOutputChannel } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { resolveIcon } from "@/lib/utils";
import { getBackgroundColorClass } from "@/utils/colors";
import { ChevronRight, GripVerticalIcon, Plus, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toTestId } from "../../utils/testID";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { ComponentBase } from "../componentBase";

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
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  canvasZoom = 1,
  disabled = false,
}: BuildingBlocksSidebarProps) {
  if (!isOpen) {
    return (
      <Button
        variant="outline"
        onClick={() => onToggle(true)}
        aria-label="Open sidebar"
        className="absolute top-4 right-4 z-10"
      >
        <Plus size={16} />
        Components
      </Button>
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

  // Close sidebar when clicking outside (for clicks in header, etc.)
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      // Don't close if we're dragging or resizing
      if (isDraggingRef.current || isResizing) {
        return;
      }

      // Don't close if clicking on the toggle button (it has its own handler)
      const target = event.target as HTMLElement;
      if (target.closest('[aria-label="Open sidebar"]')) {
        return;
      }

      if (sidebarRef.current && !sidebarRef.current.contains(event.target as Node)) {
        onToggle(false);
      }
    };

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen, onToggle, isResizing]);

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
    Components: 2,
  };

  const sortedCategories = [...(blocks || [])].sort((a, b) => {
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
      className="absolute right-0 top-0 z-20 h-full bg-white dark:bg-zinc-900 border-l border-zinc-200 dark:border-zinc-800 flex flex-col shadow-2xl"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="building-blocks-sidebar"
    >
      {/* Resize handle */}
      <div
        onMouseDown={handleMouseDown}
        className={`absolute left-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-blue-50 transition-colors flex items-center justify-center group ${
          isResizing ? "bg-blue-50" : ""
        }`}
        style={{ marginLeft: "-8px" }}
      >
        <div
          className={`w-1 h-12 rounded-full bg-gray-300 group-hover:bg-blue-500 transition-colors ${
            isResizing ? "bg-blue-500" : ""
          }`}
        />
      </div>

      {/* Header */}
      <div className="flex items-center justify-between gap-3 p-3 relative border-b-1 border-border bg-gray-50">
        <div className="flex flex-col items-start gap-3 w-full mt-2">
          <div className="flex justify-between gap-3 w-full">
            <div className="flex flex-col gap-1">
              <h2 className="text-xl font-semibold">New Component</h2>
            </div>
          </div>
          <div
            onClick={() => onToggle(false)}
            className="flex items-center justify-center absolute top-6 right-3 cursor-pointer"
          >
            <X size={18} />
          </div>
        </div>
      </div>

      {/* Search */}
      <div className="flex items-center gap-2 px-3 py-3 border-b-1 border-border">
        <div className="flex-1">
          <input
            type="text"
            placeholder="Filter components..."
            className="w-full px-3 py-2 text-sm border border-zinc-200 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder-zinc-500 dark:placeholder-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
      </div>

      <div className="flex-1 gap-2 py-3 relative">
        {sortedCategories.map((category) => (
          <CategorySection
            key={category.name}
            category={category}
            canvasZoom={canvasZoom}
            searchTerm={searchTerm}
            isDraggingRef={isDraggingRef}
            setHoveredBlock={setHoveredBlock}
            dragPreviewRef={dragPreviewRef}
          />
        ))}

        {/* Disabled overlay - only over items */}
        {disabled && (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="absolute inset-0 bg-white/60 dark:bg-zinc-900/60 z-30 cursor-not-allowed" />
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
            headerColor="bg-gray-50"
            iconSlug={hoveredBlock.icon}
            iconColor="text-indigo-700"
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
}

function CategorySection({
  category,
  canvasZoom,
  searchTerm = "",
  isDraggingRef,
  setHoveredBlock,
  dragPreviewRef,
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
    <details className="flex-1 px-5 mb-4" open>
      <summary className="cursor-pointer hover:text-zinc-600 dark:hover:text-zinc-300 mb-1 flex items-center gap-1 [&::-webkit-details-marker]:hidden [&::marker]:hidden">
        <ChevronRight className="h-3 w-3 transition-transform [[details[open]]>&]:rotate-90" />
        <span className="text-sm font-medium pl-1">{category.name}</span>
      </summary>

      <ItemGroup>
        {allBlocks.map((block) => {
          const iconSlug = block.icon || "zap";
          const IconComponent = resolveIcon(iconSlug);

          const isLive = !!block.isLive;
          return (
            <Item
              data-testid={toTestId(`building-block-${block.name}`)}
              key={`${block.type}-${block.name}`}
              draggable={isLive}
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
              className={`ml-3 px-2 py-1 flex items-center gap-2 cursor-grab active:cursor-grabbing hover:bg-zinc-50 dark:hover:bg-zinc-800/50`}
              size="sm"
            >
              <ItemMedia>
                <IconComponent size={14} className="text-indigo-700" />
              </ItemMedia>

              <ItemContent>
                <ItemTitle className="text-xs font-normal">{block.label || block.name}</ItemTitle>
              </ItemContent>

              <GripVerticalIcon className="text-zinc-500 hover:text-zinc-800" size={14} />
            </Item>
          );
        })}
      </ItemGroup>
    </details>
  );
}
