import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { resolveIcon } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import { useState } from "react";
import { toTestId } from "../../lib/testID";
import { getHeaderIconSrc, getIntegrationIconSrc } from "../componentSidebar/integrationIcons";
import type { BuildingBlock, BuildingBlockCategory } from "./types";

const TYPE_HOVER_BG: Record<string, string> = {
  trigger: "hover:bg-sky-100 dark:hover:bg-sky-900/20",
  component: "hover:bg-green-100 dark:hover:bg-green-900/20",
};

const TYPE_BADGE_COLOR: Record<string, string> = {
  trigger: "text-sky-600 dark:text-sky-400",
  component: "text-green-600 dark:text-green-400",
};

const TYPE_LABEL: Record<string, string> = {
  trigger: "Trigger",
  component: "Action",
  blueprint: "Blueprint",
};

function resolveIconSlug(block: BuildingBlock): string {
  if (block.type === "blueprint") return "component";
  const firstPart = block.name?.split(".")[0];
  if (firstPart === "smtp") return "mail";
  return block.icon || "zap";
}

interface BlockItemProps {
  block: BuildingBlock;
  onBlockClick?: (block: BuildingBlock) => void;
}

function BlockItem({ block, onBlockClick }: BlockItemProps) {
  const appIconSrc = getHeaderIconSrc(block.name);
  const IconComponent = resolveIcon(resolveIconSlug(block));
  const hoverBg = TYPE_HOVER_BG[block.type] || TYPE_HOVER_BG.component;
  const badgeColor = TYPE_BADGE_COLOR[block.type] || TYPE_BADGE_COLOR.component;

  return (
    <Item
      data-testid={toTestId(`building-block-${block.name}`)}
      onClick={() => {
        if (onBlockClick) onBlockClick(block);
      }}
      className={`ml-3 px-2 py-1 flex items-center gap-2 ${hoverBg}`}
      size="sm"
    >
      <ItemMedia>
        {appIconSrc ? (
          <img src={appIconSrc} alt={block.label || block.name} className="size-4" />
        ) : (
          <IconComponent size={14} className="text-gray-500" />
        )}
      </ItemMedia>

      <ItemContent>
        <div className="flex items-center gap-2 w-full min-w-0">
          <ItemTitle className="text-sm font-normal min-w-0 flex-1 w-0 overflow-hidden">
            <span className="block min-w-0 truncate">{block.label || block.name}</span>
          </ItemTitle>
          <span
            className={`inline-block text-left px-1.5 py-0.5 text-[11px] font-medium ${badgeColor} rounded whitespace-nowrap flex-shrink-0 ml-auto`}
          >
            {TYPE_LABEL[block.type] || "Action"}
          </span>
        </div>
      </ItemContent>
    </Item>
  );
}

export interface CategorySectionProps {
  category: BuildingBlockCategory;
  searchTerm?: string;
  onBlockClick?: (block: BuildingBlock) => void;
}

export function CategorySection({ category, searchTerm = "", onBlockClick }: CategorySectionProps) {
  const query = searchTerm.trim().toLowerCase();
  const categoryMatches = query ? (category.name || "").toLowerCase().includes(query) : true;

  const baseBlocks = categoryMatches
    ? category.blocks || []
    : (category.blocks || []).filter((block) => {
        const name = (block.name || "").toLowerCase();
        const label = (block.label || "").toLowerCase();
        return name.includes(query) || label.includes(query);
      });

  const allBlocks = baseBlocks;

  const isCoreCategory = category.name === "Core";
  const hasSearchTerm = query.length > 0;
  const [isManuallyOpen, setIsManuallyOpen] = useState<boolean | null>(null);
  const isOpen = hasSearchTerm || (isManuallyOpen ?? isCoreCategory);

  if (allBlocks.length === 0) {
    return null;
  }

  const firstBlock = allBlocks[0];
  const integrationName = firstBlock?.integrationName || category.name.toLowerCase();
  const categoryIconSrc = integrationName === "smtp" ? undefined : getIntegrationIconSrc(integrationName);

  let CategoryIcon: React.ComponentType<{ size?: number; className?: string }> | null = null;
  if (category.name === "Core") {
    CategoryIcon = resolveIcon("zap");
  } else if (category.name === "Memory") {
    CategoryIcon = resolveIcon("database");
  } else if (integrationName === "smtp") {
    CategoryIcon = resolveIcon("mail");
  } else if (categoryIconSrc) {
    // Integration category - will use img tag
  } else {
    CategoryIcon = resolveIcon("puzzle");
  }

  let sortedBlocks: BuildingBlock[] = [];
  if (isOpen) {
    const typeOrder: Record<"trigger" | "component" | "blueprint", number> = {
      trigger: 0,
      component: 1,
      blueprint: 2,
    };

    sortedBlocks = [...allBlocks].sort((a, b) => {
      const aType = a.type;
      const bType = b.type;
      const typeComparison = typeOrder[aType] - typeOrder[bType];
      if (typeComparison !== 0) {
        return typeComparison;
      }

      const aName = (a.label || a.name || "").toLowerCase();
      const bName = (b.label || b.name || "").toLowerCase();
      return aName.localeCompare(bName);
    });
  }

  return (
    <details
      className="flex-1 px-5 mb-5 group"
      open={isOpen}
      onToggle={(event) => {
        if (hasSearchTerm) {
          return;
        }
        setIsManuallyOpen(event.currentTarget.open);
      }}
    >
      <summary className="relative cursor-pointer hover:text-gray-500 dark:hover:text-gray-300 mb-3 flex w-full items-center justify-between gap-2 [&::-webkit-details-marker]:hidden [&::marker]:hidden">
        <div className="pointer-events-none absolute inset-x-0 top-1/2 -translate-y-1/2 border-t border-border/60" />
        <span className="relative z-10 flex items-center gap-1 bg-white dark:bg-gray-900 pr-3">
          <ChevronRight className="h-3 w-3 transition-transform group-open:rotate-90" />
          {categoryIconSrc ? (
            <img src={categoryIconSrc} alt={category.name} className="size-4" />
          ) : CategoryIcon ? (
            <CategoryIcon size={14} className="text-gray-500" />
          ) : null}
          <span className="text-[13px] text-gray-800 font-medium pl-1">{category.name}</span>
        </span>
      </summary>

      {isOpen && (
        <ItemGroup>
          {sortedBlocks.map((block) => (
            <BlockItem key={`${block.type}-${block.name}`} block={block} onBlockClick={onBlockClick} />
          ))}
        </ItemGroup>
      )}
    </details>
  );
}
