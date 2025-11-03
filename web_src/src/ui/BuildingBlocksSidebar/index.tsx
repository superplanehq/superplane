import type { SuperplaneBlueprintsOutputChannel, SuperplaneComponentsOutputChannel } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { resolveIcon } from "@/lib/utils";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { ChevronRight, GripVerticalIcon, Menu, PanelLeftClose, Settings2 } from "lucide-react";
import { useEffect, useState } from "react";
import { toTestId } from "../../utils/testID";
import { createNodeDragPreview } from "./createNodeDragPreview";

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
}

export function BuildingBlocksSidebar({ isOpen, onToggle, blocks, canvasZoom = 1 }: BuildingBlocksSidebarProps) {
  if (!isOpen) {
    return (
      <Button
        variant="outline"
        size="icon"
        onClick={() => onToggle(true)}
        aria-label="Open sidebar"
        className="absolute top-4 left-4 z-10 shadow-md"
      >
        <Menu size={24} />
      </Button>
    );
  }

  const [searchTerm, setSearchTerm] = useState("");
  const [isConfigOpen, setIsConfigOpen] = useState(false);

  // Initialize showWip from localStorage
  const [showWip, setShowWip] = useState(() => {
    const storedShowWip = localStorage.getItem('buildingBlocksShowWip');
    if (storedShowWip !== null) {
      return JSON.parse(storedShowWip);
    }
    return true; // default value
  });

  // Save showWip to localStorage whenever it changes
  useEffect(() => {
    localStorage.setItem('buildingBlocksShowWip', JSON.stringify(showWip));
  }, [showWip]);

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
      className="w-[360px] h-full bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col"
      data-testid="building-blocks-sidebar"
    >
      <div className="flex items-center gap-2 px-4 py-4 relative">
        <div className="flex-1">
          <input
            type="text"
            placeholder="Search components..."
            className="w-full px-3 py-2 text-sm border border-zinc-200 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder-zinc-500 dark:placeholder-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-3 pb-0">
          <Button
            variant="outline"
            size="icon"
            onClick={() => setIsConfigOpen((v) => !v)}
            aria-haspopup="menu"
            aria-expanded={isConfigOpen}
            aria-label="Configure"
          >
            <Settings2 size={20} />
          </Button>
          <Button variant="outline" size="icon" onClick={() => onToggle(false)} aria-label="Close sidebar">
            <PanelLeftClose size={24} />
          </Button>
        </div>

        {isConfigOpen && (
          <div
            role="menu"
            className="absolute right-4 top-12 z-20 w-60 rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 shadow-lg"
          >
            <button
              className="w-full text-left px-3 py-2 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
              onClick={() => {
                setShowWip((v) => !v);
                setIsConfigOpen(false);
              }}
            >
              {showWip ? "Hide WIP elements" : "Show WIP elements"}
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-scroll">
        {sortedCategories.map((category) => (
          <CategorySection
            key={category.name}
            category={category}
            canvasZoom={canvasZoom}
            showWip={showWip}
            searchTerm={searchTerm}
          />
        ))}
      </div>
    </div>
  );
}

interface CategorySectionProps {
  category: BuildingBlockCategory;
  canvasZoom: number;
  searchTerm?: string;
  showWip?: boolean;
}

function CategorySection({ category, canvasZoom, searchTerm = "", showWip = true }: CategorySectionProps) {
  const query = searchTerm.trim().toLowerCase();
  const categoryMatches = query ? (category.name || "").toLowerCase().includes(query) : true;

  const baseBlocks = categoryMatches
    ? category.blocks || []
    : (category.blocks || []).filter((block) => {
        const name = (block.name || "").toLowerCase();
        const label = (block.label || "").toLowerCase();
        return name.includes(query) || label.includes(query);
      });

  const allBlocks = showWip ? baseBlocks : baseBlocks.filter((b) => b.isLive);

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
          const colorClass = getColorClass(block.color);
          const backgroundColorClass = getBackgroundColorClass(block.color);

          const isLive = !!block.isLive;
          return (
            <Item
              data-testid={toTestId(`building-block-${block.name}`)}
              key={`${block.type}-${block.name}`}
              draggable={isLive}
              onDragStart={(e) => {
                if (!isLive) {
                  e.preventDefault();
                  return;
                }
                createNodeDragPreview(e, block, colorClass, backgroundColorClass, canvasZoom);
              }}
              aria-disabled={!isLive}
              title={isLive ? undefined : "Coming soon"}
              className={`ml-3 px-2 py-1 flex items-center gap-2 cursor-grab active:cursor-grabbing hover:bg-zinc-50 dark:hover:bg-zinc-800/50`}
              size="sm"
            >
              <ItemMedia>
                <IconComponent size={14} className={colorClass} />
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
