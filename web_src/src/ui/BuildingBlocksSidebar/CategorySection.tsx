import type { OrganizationsIntegration } from "@/api-client";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { resolveIcon } from "@/lib/utils";
import { ChevronRight, GripVerticalIcon, Plug } from "lucide-react";
import { useState } from "react";
import { toTestId } from "../../lib/testID";
import { getComponentSubtype } from "../buildingBlocks";
import { getHeaderIconSrc, getIntegrationIconSrc } from "../componentSidebar/integrationIcons";
import { BuildingBlockPreview } from "./BuildingBlockPreview";
import type { BuildingBlock, BuildingBlockCategory } from "./types";

const SUBTYPE_HOVER_BG: Record<string, string> = {
  trigger: "hover:bg-sky-100 dark:hover:bg-sky-900/20",
  flow: "hover:bg-purple-100 dark:hover:bg-purple-900/20",
  action: "hover:bg-green-100 dark:hover:bg-green-900/20",
};

const SUBTYPE_BADGE_COLOR: Record<string, string> = {
  trigger: "text-sky-600 dark:text-sky-400",
  flow: "text-purple-600 dark:text-purple-400",
  action: "text-green-600 dark:text-green-400",
};

const SUBTYPE_LABEL: Record<string, string> = {
  trigger: "Trigger",
  flow: "Flow",
  action: "Action",
};

function resolveIconSlug(block: BuildingBlock): string {
  if (block.type === "blueprint") return "component";
  const firstPart = block.name?.split(".")[0];
  if (firstPart === "smtp") return "mail";
  return block.icon || "zap";
}

function setupDragPreview(
  e: React.DragEvent,
  dragPreviewRef: React.RefObject<HTMLDivElement | null>,
  canvasZoom: number,
) {
  const previewElement = dragPreviewRef.current?.firstChild as HTMLElement;
  if (!previewElement) return;

  const clone = previewElement.cloneNode(true) as HTMLElement;
  const container = document.createElement("div");
  container.style.cssText = `position: absolute; top: -10000px; left: -10000px; pointer-events: none;`;
  clone.style.transform = `scale(${canvasZoom})`;
  clone.style.transformOrigin = "top left";
  clone.style.opacity = "0.85";
  container.appendChild(clone);
  document.body.appendChild(container);

  const rect = previewElement.getBoundingClientRect();
  e.dataTransfer.setDragImage(container, (rect.width / 2) * canvasZoom, 30 * canvasZoom);
  setTimeout(() => {
    if (document.body.contains(container)) document.body.removeChild(container);
  }, 0);
}

interface BlockItemProps {
  block: BuildingBlock;
  canvasZoom: number;
  isDraggingRef: React.RefObject<boolean>;
  setHoveredBlock: (block: BuildingBlock | null) => void;
  dragPreviewRef: React.RefObject<HTMLDivElement | null>;
  onBlockClick?: (block: BuildingBlock) => void;
}

function BlockItem({
  block,
  canvasZoom,
  isDraggingRef,
  setHoveredBlock,
  dragPreviewRef,
  onBlockClick,
}: BlockItemProps) {
  const appIconSrc = getHeaderIconSrc(block.name);
  const IconComponent = resolveIcon(resolveIconSlug(block));
  const subtype = block.componentSubtype || getComponentSubtype(block);
  const hoverBg = SUBTYPE_HOVER_BG[subtype] || SUBTYPE_HOVER_BG.action;
  const badgeColor = SUBTYPE_BADGE_COLOR[subtype] || SUBTYPE_BADGE_COLOR.action;

  return (
    <BuildingBlockPreview block={block}>
      <Item
        data-testid={toTestId(`building-block-${block.name}`)}
        draggable
        onClick={() => {
          if (onBlockClick) onBlockClick(block);
        }}
        onMouseEnter={() => {
          setHoveredBlock(block);
        }}
        onMouseLeave={() => {
          setHoveredBlock(null);
        }}
        onDragStart={(e) => {
          isDraggingRef.current = true;
          e.dataTransfer.effectAllowed = "move";
          e.dataTransfer.setData("application/reactflow", JSON.stringify(block));
          setupDragPreview(e, dragPreviewRef, canvasZoom);
        }}
        onDragEnd={() => {
          isDraggingRef.current = false;
          setHoveredBlock(null);
        }}
        className={`ml-3 px-2 py-1 flex items-center gap-2 cursor-grab active:cursor-grabbing ${hoverBg}`}
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
              {SUBTYPE_LABEL[subtype] || "Action"}
            </span>
            {block.deprecated && (
              <span className="px-1.5 py-0.5 text-[11px] font-medium bg-gray-950/5 text-gray-500 rounded whitespace-nowrap flex-shrink-0">
                Deprecated
              </span>
            )}
          </div>
        </ItemContent>

        <GripVerticalIcon className="text-gray-500 hover:text-gray-800" size={14} />
      </Item>
    </BuildingBlockPreview>
  );
}

export interface CategorySectionProps {
  category: BuildingBlockCategory;
  integrations: OrganizationsIntegration[];
  showIntegrationSetupStatus: boolean;
  canvasZoom: number;
  searchTerm?: string;
  typeFilter?: "all" | "trigger" | "action" | "flow";
  isDraggingRef: React.RefObject<boolean>;
  setHoveredBlock: (block: BuildingBlock | null) => void;
  dragPreviewRef: React.RefObject<HTMLDivElement | null>;
  onBlockClick?: (block: BuildingBlock) => void;
}

export function CategorySection({
  category,
  integrations,
  showIntegrationSetupStatus,
  canvasZoom,
  searchTerm = "",
  typeFilter = "all",
  isDraggingRef,
  setHoveredBlock,
  dragPreviewRef,
  onBlockClick,
}: CategorySectionProps) {
  const normalizeIntegrationName = (value?: string) => (value || "").toLowerCase().replace(/[^a-z0-9]/g, "");

  const query = searchTerm.trim().toLowerCase();
  const categoryMatches = query ? (category.name || "").toLowerCase().includes(query) : true;

  const baseBlocks = categoryMatches
    ? category.blocks || []
    : (category.blocks || []).filter((block) => {
        const name = (block.name || "").toLowerCase();
        const label = (block.label || "").toLowerCase();
        return name.includes(query) || label.includes(query);
      });

  let allBlocks = baseBlocks;

  if (typeFilter !== "all") {
    allBlocks = allBlocks.filter((block) => {
      const subtype = block.componentSubtype || getComponentSubtype(block);
      return subtype === typeFilter;
    });
  }

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

  const normalizedIntegrationName = normalizeIntegrationName(firstBlock?.integrationName);
  const matchingIntegrationStates = normalizedIntegrationName
    ? integrations
        .filter(
          (integration) => normalizeIntegrationName(integration.spec?.integrationName) === normalizedIntegrationName,
        )
        .map((integration) => integration.status?.state)
    : [];

  const integrationState =
    category.name === "Core" || category.name === "Memory"
      ? "ready"
      : matchingIntegrationStates.includes("ready")
        ? "ready"
        : matchingIntegrationStates.includes("error")
          ? "error"
          : matchingIntegrationStates.includes("pending")
            ? "pending"
            : undefined;

  const integrationStatusColorClass =
    integrationState === "ready"
      ? "text-green-500"
      : integrationState === "error"
        ? "text-red-500"
        : integrationState === "pending"
          ? "text-amber-600"
          : "text-gray-500";

  let CategoryIcon: React.ComponentType<{ size?: number; className?: string }> | null = null;
  if (category.name === "Core") {
    CategoryIcon = resolveIcon("zap");
  } else if (category.name === "Memory") {
    CategoryIcon = resolveIcon("database");
  } else if (category.name === "Bundles") {
    CategoryIcon = resolveIcon("package");
  } else if (integrationName === "smtp") {
    CategoryIcon = resolveIcon("mail");
  } else if (categoryIconSrc) {
    // Integration category - will use img tag
  } else {
    CategoryIcon = resolveIcon("puzzle");
  }

  let sortedBlocks: BuildingBlock[] = [];
  if (isOpen) {
    const subtypeOrder: Record<"trigger" | "action" | "flow", number> = {
      trigger: 0,
      action: 1,
      flow: 2,
    };

    sortedBlocks = [...allBlocks].sort((a, b) => {
      const aSubtype = a.componentSubtype || getComponentSubtype(a);
      const bSubtype = b.componentSubtype || getComponentSubtype(b);
      const subtypeComparison = subtypeOrder[aSubtype] - subtypeOrder[bSubtype];
      if (subtypeComparison !== 0) {
        return subtypeComparison;
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
        {showIntegrationSetupStatus && (
          <span className="relative z-10 shrink-0 bg-white dark:bg-gray-900 pl-3">
            <Plug size={14} className={integrationStatusColorClass} />
          </span>
        )}
      </summary>

      {isOpen && (
        <ItemGroup>
          {sortedBlocks.map((block) => (
            <BlockItem
              key={`${block.type}-${block.name}`}
              block={block}
              canvasZoom={canvasZoom}
              isDraggingRef={isDraggingRef}
              setHoveredBlock={setHoveredBlock}
              dragPreviewRef={dragPreviewRef}
              onBlockClick={onBlockClick}
            />
          ))}
        </ItemGroup>
      )}
    </details>
  );
}
