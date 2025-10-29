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
import { getColorClass } from "@/utils/colors";
import { ChevronRight, Menu, PanelLeftClose } from "lucide-react";
import { useState } from "react";

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
  onBlockClick: (block: BuildingBlock) => void;
  blocks: BuildingBlockCategory[];
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  onBlockClick,
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

  const sortedCategories = blocks.sort((a, b) => {
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
            onBlockClick={onBlockClick}
          />
        ))}
      </div>
    </div>
  );
}

const getColor = (color?: string): string => {
  switch (color) {
    case "blue":
      return "bg-blue-200";
    case "green":
      return "bg-green-200";
    case "red":
      return "bg-red-200";
    case "yellow":
      return "bg-yellow-200";
    case "purple":
      return "bg-purple-200";
    case "orange":
      return "bg-orange-200";
    case "pink":
      return "bg-pink-200";
    case "indigo":
      return "bg-indigo-200";
    case "sky":
      return "bg-sky-200";
    case "gray":
      return "bg-gray-200";
    default:
      return "bg-gray-200";
  }
};

interface CategorySectionProps {
  category: BuildingBlockCategory;
  onBlockClick: (block: BuildingBlock) => void;
}

function CategorySection({ category, onBlockClick }: CategorySectionProps) {
  const allBlocks = category.blocks;
  const isPrimitives = category.name === "Primitives";
  const categoryColor = getColorClass(allBlocks[0]?.color);
  const categoryBg = getColor(allBlocks[0]?.color);

  return (
    <details className="flex-1 px-5 mb-4" open>
      <summary className="cursor-pointer hover:text-zinc-600 dark:hover:text-zinc-300 mb-1 flex items-center gap-1 [&::-webkit-details-marker]:hidden [&::marker]:hidden">
        <ChevronRight className="h-3 w-3 transition-transform [[details[open]]>&]:rotate-90" />
        <span className="text-sm font-medium pl-1">{category.name}</span>
      </summary>

      <ItemGroup>
        {allBlocks.map((block) => {
          const iconSlug = isPrimitives ? block.icon || "zap" : undefined;
          const IconComponent = resolveIcon(iconSlug);
          const colorClass = isPrimitives
            ? getColorClass(block.color)
            : categoryColor;

          return (
            <Item
              key={`${block.type}-${block.name}`}
              onClick={() => onBlockClick(block)}
              className="ml-3 cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50 px-2 py-1 flex items-center gap-2"
              size="sm"
            >
              <ItemMedia>
                {isPrimitives ? (
                  <IconComponent size={14} className={colorClass} />
                ) : (
                  <span
                    className={`inline-block h-[14px] w-[14px] rounded-[4px] ${categoryBg}`}
                  />
                )}
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
