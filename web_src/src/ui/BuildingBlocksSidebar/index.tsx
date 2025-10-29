import type {
  SuperplaneBlueprintsOutputChannel,
  SuperplaneComponentsOutputChannel,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Item,
  ItemContent,
  ItemDescription,
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

export interface BuildingBlocksSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  triggers: BuildingBlock[];
  components: BuildingBlock[];
  blueprints: BuildingBlock[];
  onBlockClick: (block: BuildingBlock) => void;
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  triggers,
  components,
  blueprints,
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

  const allBlocks = triggers.concat(components).concat(blueprints);

  return (
    <div className="w-96 h-full bg-white dark:bg-zinc-900 border-r border-zinc-200 dark:border-zinc-800 flex flex-col">
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

      <div className="flex-1 overflow-hidden px-4 pt-4">
        <ItemGroup>
          {allBlocks.map((block) => {
            const IconComponent = resolveIcon(block.icon || "zap");
            const colorClass = getColorClass(block.color);

            return (
              <Item
                key={`${block.type}-${block.name}`}
                onClick={() => onBlockClick(block)}
                className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                size="sm"
              >
                <ItemMedia>
                  <IconComponent size={24} className={colorClass} />
                </ItemMedia>
                <ItemContent>
                  <ItemTitle>{block.label || block.name}</ItemTitle>
                  {block.description && (
                    <ItemDescription>{block.description}</ItemDescription>
                  )}
                </ItemContent>
              </Item>
            );
          })}
        </ItemGroup>
      </div>
    </div>
  );
}
