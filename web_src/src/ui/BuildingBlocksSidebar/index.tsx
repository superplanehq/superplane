import type {
  SuperplaneBlueprintsOutputChannel,
  SuperplaneComponentsOutputChannel,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Item,
  ItemContent,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import { resolveIcon } from "@/lib/utils";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";
import { ChevronRight, Menu, PanelLeftClose } from "lucide-react";
import { useState } from "react";
import { createNodeDragPreview } from "./createNodeDragPreview";

export interface BuildingBlock {
  name: string;
  label?: string;
  description?: string;
  type: "trigger" | "component" | "blueprint";
  outputChannels?: Array<
    SuperplaneComponentsOutputChannel | SuperplaneBlueprintsOutputChannel
  >;
  configuration?: any[];
  icon?: string;
  color?: string;
  id?: string; // for blueprints
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

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  canvasZoom = 1,
}: BuildingBlocksSidebarProps) {
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

  const sortedCategories = (blocks || []).sort((a, b) => {
    if (a.name === "Primitives") return -1;
    if (b.name === "Primitives") return 1;
    if (a.name === "Custom Components") return -1;
    if (b.name === "Custom Components") return 1;
    return a.name.localeCompare(b.name);
  });

  return (
    <div className="w-[300px] h-full bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col">
      <div className="flex items-center gap-2 px-4 py-4">
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
            onClick={() => onToggle(false)}
            aria-label="Close sidebar"
          >
            <PanelLeftClose size={24} />
          </Button>
        </div>
      </div>

      <div className="flex-1 overflow-y-scroll">
        {sortedCategories.map((category) => (
          <CategorySection
            key={category.name}
            category={category}
            canvasZoom={canvasZoom}
          />
        ))}
      </div>
    </div>
  );
}

interface CategorySectionProps {
  category: BuildingBlockCategory;
  canvasZoom: number;
}

function CategorySection({ category, canvasZoom }: CategorySectionProps) {
  const allBlocks = category.blocks;

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

          return (
            <Item
              key={`${block.type}-${block.name}`}
              draggable
              onDragStart={(e) =>
                createNodeDragPreview(
                  e,
                  block,
                  colorClass,
                  backgroundColorClass,
                  canvasZoom
                )
              }
              className="ml-3 cursor-grab active:cursor-grabbing hover:bg-zinc-50 dark:hover:bg-zinc-800/50 px-2 py-1 flex items-center gap-2"
              size="sm"
            >
              <ItemMedia>
                <IconComponent size={14} className={colorClass} />
              </ItemMedia>

              <ItemContent>
                <ItemTitle className="text-xs font-normal">
                  {block.label || block.name}
                </ItemTitle>
              </ItemContent>
            </Item>
          );
        })}
      </ItemGroup>
    </details>
  );
}
