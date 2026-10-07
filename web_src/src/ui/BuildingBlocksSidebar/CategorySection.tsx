import type { OrganizationsIntegration } from "@/api-client";
import SuperplaneLogo from "@/assets/superplane.svg";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { cn, resolveIcon } from "@/lib/utils";
import {
  factorySidebarKindLabelClassName,
  factorySidebarMutedIconClassName,
  factorySidebarRowHoverClassName,
} from "@/ui/factoryNodeChrome";
import { ChevronRight, Plug } from "lucide-react";
import { memo, useState, type DragEvent } from "react";
import { toTestId } from "../../lib/testID";
import { getHeaderIconSrc, getIntegrationIconSrc } from "../componentSidebar/integrationIconMaps";
import { filterBlocksInCategory, normalizeIntegrationName, type TypeFilter } from "./filter";
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
};

function resolveIconSlug(block: BuildingBlock): string {
  const firstPart = block.name?.split(".")[0];
  if (firstPart === "smtp") return "mail";
  return block.icon || "zap";
}

function resolveCategoryIcon(categoryName: string, integrationName: string) {
  if (categoryName === "Core") {
    return resolveIcon("zap");
  }
  if (categoryName === "Runners") {
    return resolveIcon("terminal");
  }
  if (categoryName === "Debugging") {
    return resolveIcon("bug");
  }
  if (categoryName === "Memory") {
    return resolveIcon("database");
  }
  if (integrationName === "smtp") {
    return resolveIcon("mail");
  }
  return resolveIcon("puzzle");
}

function renderCategoryIcon(
  categoryIconSrc: string | undefined,
  categoryName: string,
  CategoryIcon: React.ComponentType<{ size?: number; className?: string }> | null,
  iconClassName: string,
) {
  if (categoryIconSrc) {
    return (
      <img
        src={categoryIconSrc}
        alt={categoryName}
        className={categoryName === "SuperPlane" ? "size-4 dark:brightness-0 dark:invert" : "size-4"}
      />
    );
  }
  if (CategoryIcon) {
    return <CategoryIcon size={14} className={iconClassName} />;
  }
  return null;
}

interface BlockItemProps {
  block: BuildingBlock;
  onBlockClick?: (block: BuildingBlock) => void;
  factoryChrome?: boolean;
}

const BlockItem = memo(function BlockItem({ block, onBlockClick, factoryChrome = false }: BlockItemProps) {
  const appIconSrc = getHeaderIconSrc(block.name);
  const IconComponent = resolveIcon(resolveIconSlug(block));
  const hoverBg = factoryChrome
    ? factorySidebarRowHoverClassName
    : TYPE_HOVER_BG[block.type] || TYPE_HOVER_BG.component;
  const badgeColor = factoryChrome
    ? factorySidebarKindLabelClassName(block.type)
    : TYPE_BADGE_COLOR[block.type] || TYPE_BADGE_COLOR.component;
  const iconClassName = factoryChrome ? factorySidebarMutedIconClassName : "text-gray-500 dark:text-gray-400";

  return (
    <Item
      data-testid={toTestId(`building-block-${block.name}`)}
      draggable
      onClick={() => {
        if (onBlockClick) onBlockClick(block);
      }}
      onDragStart={(event: DragEvent<HTMLElement>) => {
        event.dataTransfer.effectAllowed = "move";
        event.dataTransfer.setData("application/reactflow", JSON.stringify(block));
      }}
      className={`ml-3 flex cursor-pointer items-center gap-2 px-2 py-1 ${hoverBg}`}
      size="sm"
    >
      <ItemMedia>
        {appIconSrc ? (
          <img src={appIconSrc} alt={block.label || block.name} className="size-3.5" />
        ) : (
          <IconComponent size={14} className={iconClassName} />
        )}
      </ItemMedia>

      <ItemContent>
        <div className="flex w-full min-w-0 items-center gap-2">
          <ItemTitle
            className={cn(
              "min-w-0 w-0 flex-1 overflow-hidden text-[13px] font-normal",
              factoryChrome && "text-foreground",
            )}
          >
            <span className="block min-w-0 truncate">{block.label || block.name}</span>
          </ItemTitle>
          <span
            className={cn(
              "ml-auto inline-block shrink-0 whitespace-nowrap rounded px-1.5 py-0.5 text-left",
              badgeColor,
            )}
          >
            {TYPE_LABEL[block.type] || "Action"}
          </span>
        </div>
      </ItemContent>
    </Item>
  );
});

export interface CategorySectionProps {
  category: BuildingBlockCategory;
  integrations?: OrganizationsIntegration[];
  showIntegrationSetupStatus?: boolean;
  searchTerm?: string;
  typeFilter?: TypeFilter;
  onBlockClick?: (block: BuildingBlock) => void;
  factoryChrome?: boolean;
}

type IntegrationState = "ready" | "error" | "pending" | "notConfigured";

const INTEGRATION_STATE_COLOR: Record<IntegrationState, string> = {
  ready: "text-green-500",
  error: "text-red-500",
  pending: "text-amber-600",
  notConfigured: "text-gray-500",
};

function factoryOrOrgClass(factoryChrome: boolean, factoryClass: string, orgClass: string): string {
  if (factoryChrome) {
    return factoryClass;
  }
  return orgClass;
}

function resolveCategoryIconSrc(categoryName: string, integrationName: string): string | undefined {
  if (categoryName === "SuperPlane") {
    return SuperplaneLogo;
  }
  if (integrationName === "smtp") {
    return undefined;
  }
  return getIntegrationIconSrc(integrationName);
}

export function CategorySection({
  category,
  integrations = [],
  showIntegrationSetupStatus = false,
  searchTerm = "",
  typeFilter = "all",
  onBlockClick,
  factoryChrome = false,
}: CategorySectionProps) {
  const sortedBlocks = filterBlocksInCategory(category, searchTerm, typeFilter);

  const isCoreCategory = category.name === "Core";
  const hasSearchTerm = searchTerm.trim().length > 0;
  const [isManuallyOpen, setIsManuallyOpen] = useState<boolean | null>(null);
  const isOpen = hasSearchTerm || (isManuallyOpen ?? isCoreCategory);

  if (sortedBlocks.length === 0) {
    return null;
  }

  const firstBlock = sortedBlocks[0];
  const integrationName = firstBlock?.integrationName || category.name.toLowerCase();
  const categoryIconSrc = resolveCategoryIconSrc(category.name, integrationName);
  const CategoryIcon = categoryIconSrc ? null : resolveCategoryIcon(category.name, integrationName);
  const iconClassName = factoryOrOrgClass(
    factoryChrome,
    factorySidebarMutedIconClassName,
    "text-gray-500 dark:text-gray-400",
  );
  const surfaceClass = factoryOrOrgClass(factoryChrome, "bg-background", "bg-white dark:bg-gray-900");
  const integrationStatusColorClass =
    INTEGRATION_STATE_COLOR[resolveIntegrationState(category, integrations, firstBlock)];

  return (
    <details
      className="flex-1 px-5 mb-5 group"
      open={isOpen}
      onToggle={(event) => {
        if (!hasSearchTerm) {
          setIsManuallyOpen(event.currentTarget.open);
        }
      }}
    >
      <summary
        className={cn(
          "relative mb-3 flex w-full cursor-pointer items-center justify-between gap-2 [&::-webkit-details-marker]:hidden [&::marker]:hidden",
          factoryOrOrgClass(
            factoryChrome,
            "hover:text-muted-foreground",
            "hover:text-gray-500 dark:hover:text-gray-300",
          ),
        )}
      >
        <div className="pointer-events-none absolute inset-x-0 top-1/2 -translate-y-1/2 border-t border-border/60" />
        <span className={cn("relative z-10 flex items-center gap-1 pr-3", surfaceClass)}>
          <ChevronRight className="h-3 w-3 transition-transform group-open:rotate-90" />
          {renderCategoryIcon(categoryIconSrc, category.name, CategoryIcon, iconClassName)}
          <span
            className={cn(
              "pl-1 text-[13px] font-medium",
              factoryOrOrgClass(factoryChrome, "text-foreground", "text-gray-800 dark:text-gray-100"),
            )}
          >
            {category.name}
          </span>
        </span>
        {showIntegrationSetupStatus && (
          <span className={cn("relative z-10 shrink-0 pl-3", surfaceClass)}>
            <Plug size={14} className={integrationStatusColorClass} />
          </span>
        )}
      </summary>

      <ItemGroup>
        {sortedBlocks.map((block) => (
          <BlockItem
            key={`${block.type}-${block.name}`}
            block={block}
            onBlockClick={onBlockClick}
            factoryChrome={factoryChrome}
          />
        ))}
      </ItemGroup>
    </details>
  );
}

function resolveIntegrationState(
  category: BuildingBlockCategory,
  integrations: OrganizationsIntegration[],
  firstBlock: BuildingBlock,
): IntegrationState {
  if (
    category.name === "Core" ||
    category.name === "SuperPlane" ||
    category.name === "Memory" ||
    category.name === "Debugging"
  ) {
    return "ready";
  }

  const name = normalizeIntegrationName(firstBlock?.integrationName);
  const matchingStates = integrations
    .filter((integration) => normalizeIntegrationName(integration.metadata?.integrationName) === name)
    .map((integration) => integration.status?.state);

  if (matchingStates.includes("ready")) {
    return "ready";
  }
  if (matchingStates.includes("error")) {
    return "error";
  }
  if (matchingStates.includes("pending")) {
    return "pending";
  }
  return "notConfigured";
}
