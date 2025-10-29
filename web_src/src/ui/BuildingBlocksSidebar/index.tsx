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
import { Menu, PanelLeftClose } from "lucide-react";

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

  return (
    <div className="w-[280px] h-full bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col">
      <div className="flex items-center gap-3 px-4 pt-4 pb-0">
        <Button
          variant="outline"
          size="icon"
          onClick={() => onToggle(false)}
          aria-label="Close sidebar"
        >
          <PanelLeftClose size={24} />
        </Button>
      </div>

      <div className="flex-1 overflow-y-scroll">
        {blocks.map((category) => (
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

interface CategorySectionProps {
  category: BuildingBlockCategory;
  onBlockClick: (block: BuildingBlock) => void;
}

function CategorySection({ category, onBlockClick }: CategorySectionProps) {
  const allBlocks = category.blocks;

  return (
    <details className="flex-1 px-4 pt-4" open>
      <summary className="text-xs uppercase cursor-pointer hover:text-zinc-600 dark:hover:text-zinc-300 mb-2">
        <span className="pl-2">{category.name}</span>
      </summary>

      <ItemGroup>
        {allBlocks.map((block) => {
          const IconComponent = resolveIcon(block.icon || "zap");
          const colorClass = getColorClass(block.color);

          return (
            <Item
              key={`${block.type}-${block.name}`}
              onClick={() => onBlockClick(block)}
              className="ml-3 cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50 px-2 py-1"
              size="sm"
            >
              <ItemMedia>
                <IconComponent size={18} className={colorClass} />
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
