import type { SuperplaneBlueprintsOutputChannel, SuperplaneComponentsOutputChannel } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { resolveIcon } from "@/lib/utils";
import { isCustomComponentsEnabled } from "@/lib/env";
import { getBackgroundColorClass } from "@/utils/colors";
import { getComponentSubtype } from "../buildingBlocks";
import { ChevronRight, GripVerticalIcon, Plus, Search, StickyNote, X } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toTestId } from "../../utils/testID";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { ComponentBase } from "../componentBase";
import circleciIcon from "@/assets/icons/integrations/circleci.svg";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import daytonaIcon from "@/assets/icons/integrations/daytona.svg";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import discordIcon from "@/assets/icons/integrations/discord.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import cursorIcon from "@/assets/icons/integrations/cursor.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import awsLambdaIcon from "@/assets/icons/integrations/aws.lambda.svg";
import awsRoute53Icon from "@/assets/icons/integrations/aws.route53.svg";
import awsEcrIcon from "@/assets/icons/integrations/aws.ecr.svg";
import awsEcsIcon from "@/assets/icons/integrations/aws.ecs.svg";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import awsSnsIcon from "@/assets/icons/integrations/aws.sns.svg";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import sendgridIcon from "@/assets/icons/integrations/sendgrid.svg";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import renderIcon from "@/assets/icons/integrations/render.svg";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import hetznerIcon from "@/assets/icons/integrations/hetzner.svg";

export interface BuildingBlock {
  name: string;
  label?: string;
  description?: string;
  type: "trigger" | "component" | "blueprint";
  componentSubtype?: "trigger" | "action" | "flow";
  outputChannels?: Array<SuperplaneComponentsOutputChannel | SuperplaneBlueprintsOutputChannel>;
  configuration?: any[];
  icon?: string;
  color?: string;
  id?: string; // for blueprints
  isLive?: boolean; // marks items that actually work now
  integrationName?: string; // for components/triggers from integrations
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
  disabledMessage?: string;
  onBlockClick?: (block: BuildingBlock) => void;
  onAddNote?: () => void;
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  canvasZoom = 1,
  disabled = false,
  disabledMessage,
  onBlockClick,
  onAddNote,
}: BuildingBlocksSidebarProps) {
  const disabledTooltip = disabledMessage || "Finish configuring the selected component first";

  if (!isOpen) {
    const addNoteButton = (
      <Button
        variant="outline"
        onClick={() => {
          if (disabled) return;
          onAddNote?.();
        }}
        aria-label="Add Note"
        data-testid="add-note-button"
        disabled={disabled}
      >
        <StickyNote size={16} className="animate-pulse" />
        Add Note
      </Button>
    );
    const openSidebarButton = (
      <Button
        variant="outline"
        onClick={() => {
          if (disabled) return;
          onToggle(true);
        }}
        aria-label="Open sidebar"
        data-testid="open-sidebar-button"
        disabled={disabled}
      >
        <Plus size={16} />
        Components
      </Button>
    );

    return (
      <div className="absolute top-4 right-4 z-10 flex gap-3">
        {disabled ? (
          <Tooltip>
            <TooltipTrigger asChild>{addNoteButton}</TooltipTrigger>
            <TooltipContent side="left" sideOffset={10}>
              <p>{disabledTooltip}</p>
            </TooltipContent>
          </Tooltip>
        ) : (
          addNoteButton
        )}
        {disabled ? (
          <Tooltip>
            <TooltipTrigger asChild>{openSidebarButton}</TooltipTrigger>
            <TooltipContent side="left" sideOffset={10}>
              <p>{disabledTooltip}</p>
            </TooltipContent>
          </Tooltip>
        ) : (
          openSidebarButton
        )}
      </div>
    );
  }

  const [searchTerm, setSearchTerm] = useState("");
  const [typeFilter, setTypeFilter] = useState<"all" | "trigger" | "action" | "flow">("all");
  const sidebarRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
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

  // Auto-focus search input when sidebar opens
  useEffect(() => {
    if (isOpen && searchInputRef.current) {
      // Small delay to ensure the sidebar is fully rendered
      setTimeout(() => {
        searchInputRef.current?.focus();
      }, 100);
    }
  }, [isOpen]);

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
    Core: 0,
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
              <h2 className="text-base font-medium">New Component</h2>
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

      {/* Search and Filter */}
      <div className="flex items-center gap-2 px-5">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" size={16} />
          <Input
            ref={searchInputRef}
            type="text"
            placeholder="Filter components..."
            className="pl-9"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
        <Select value={typeFilter} onValueChange={(value) => setTypeFilter(value as typeof typeFilter)}>
          <SelectTrigger size="sm" className="w-[120px]">
            <SelectValue placeholder="All Types" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Types</SelectItem>
            <SelectItem value="trigger">Trigger</SelectItem>
            <SelectItem value="action">Action</SelectItem>
            <SelectItem value="flow">Flow</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="gap-2 py-6">
        {sortedCategories.map((category) => (
          <CategorySection
            key={category.name}
            category={category}
            canvasZoom={canvasZoom}
            searchTerm={searchTerm}
            typeFilter={typeFilter}
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
              <p>{disabledTooltip}</p>
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
            iconSlug={hoveredBlock.name?.split(".")[0] === "smtp" ? "mail" : (hoveredBlock.icon ?? "zap")}
            iconColor="text-gray-800"
            collapsedBackground={getBackgroundColorClass("white")}
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
  typeFilter?: "all" | "trigger" | "action" | "flow";
  isDraggingRef: React.RefObject<boolean>;
  setHoveredBlock: (block: BuildingBlock | null) => void;
  dragPreviewRef: React.RefObject<HTMLDivElement | null>;
  onBlockClick?: (block: BuildingBlock) => void;
}

function CategorySection({
  category,
  canvasZoom,
  searchTerm = "",
  typeFilter = "all",
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
  let allBlocks = baseBlocks.filter((b) => b.isLive);

  // Apply type filter
  if (typeFilter !== "all") {
    allBlocks = allBlocks.filter((block) => {
      const subtype = block.componentSubtype || getComponentSubtype(block);
      return subtype === typeFilter;
    });
  }

  if (allBlocks.length === 0) {
    return null;
  }

  // Determine category icon
  const appLogoMap: Record<string, string | Record<string, string>> = {
    bitbucket: bitbucketIcon,
    circleci: circleciIcon,
    cloudflare: cloudflareIcon,
    dash0: dash0Icon,
    datadog: datadogIcon,
    daytona: daytonaIcon,
    discord: discordIcon,
    github: githubIcon,
    gitlab: gitlabIcon,
    hetzner: hetznerIcon,
    jira: jiraIcon,
    openai: openAiIcon,
    "open-ai": openAiIcon,
    claude: claudeIcon,
    cursor: cursorIcon,
    pagerduty: pagerDutyIcon,
    rootly: rootlyIcon,
    semaphore: SemaphoreLogo,
    slack: slackIcon,
    sendgrid: sendgridIcon,
    prometheus: prometheusIcon,
    render: renderIcon,
    dockerhub: dockerIcon,
    aws: {
      codeArtifact: awsIcon,
      cloudwatch: awsCloudwatchIcon,
      lambda: awsLambdaIcon,
      ecr: awsEcrIcon,
      route53: awsRoute53Icon,
      ecs: awsEcsIcon,
      sns: awsSnsIcon,
    },
  };

  // Get integration name from first block if available, or match category name
  const firstBlock = allBlocks[0];
  const integrationName = firstBlock?.integrationName || category.name.toLowerCase();
  const appLogo = appLogoMap[integrationName];
  const categoryIconSrc = typeof appLogo === "string" ? appLogo : integrationName === "aws" ? awsIcon : undefined;

  // Determine icon for special categories (Core, Bundles, SMTP use Lucide SVG; others use img when categoryIconSrc)
  let CategoryIcon: React.ComponentType<{ size?: number; className?: string }> | null = null;
  if (category.name === "Core") {
    CategoryIcon = resolveIcon("zap");
  } else if (category.name === "Bundles") {
    CategoryIcon = resolveIcon("package");
  } else if (integrationName === "smtp") {
    CategoryIcon = resolveIcon("mail");
  } else if (categoryIconSrc) {
    // Integration category - will use img tag
  } else {
    CategoryIcon = resolveIcon("puzzle");
  }

  const isCoreCategory = category.name === "Core";
  const hasSearchTerm = query.length > 0;
  // Expand if it's Core category (default) or if there's a search term (show results)
  const shouldBeOpen = isCoreCategory || hasSearchTerm;

  return (
    <details className="flex-1 px-5 mb-5 group" open={shouldBeOpen}>
      <summary className="relative cursor-pointer hover:text-gray-500 dark:hover:text-gray-300 mb-3 flex items-center gap-1 [&::-webkit-details-marker]:hidden [&::marker]:hidden">
        <span className="relative z-10 flex items-center gap-1 bg-white dark:bg-gray-900 pr-3">
          <ChevronRight className="h-3 w-3 transition-transform group-open:rotate-90" />
          {categoryIconSrc ? (
            <img src={categoryIconSrc} alt={category.name} className="size-3.5" />
          ) : CategoryIcon ? (
            <CategoryIcon size={14} className="text-gray-500" />
          ) : null}
          <span className="text-[13px] text-gray-500 font-medium pl-1">
            {category.name} ({allBlocks.length})
          </span>
        </span>
      </summary>

      <ItemGroup>
        {allBlocks.map((block) => {
          const nameParts = block.name?.split(".") ?? [];
          const iconSlug =
            block.type === "blueprint" ? "component" : nameParts[0] === "smtp" ? "mail" : block.icon || "zap";

          // Use SVG icons for application components/triggers (SMTP uses resolveIcon("mail"), same as core)
          const appLogoMap: Record<string, string | Record<string, string>> = {
            bitbucket: bitbucketIcon,
            circleci: circleciIcon,
            cloudflare: cloudflareIcon,
            dash0: dash0Icon,
            daytona: daytonaIcon,
            datadog: datadogIcon,
            discord: discordIcon,
            github: githubIcon,
            gitlab: gitlabIcon,
            hetzner: hetznerIcon,
            openai: openAiIcon,
            "open-ai": openAiIcon,
            claude: claudeIcon,
            cursor: cursorIcon,
            pagerduty: pagerDutyIcon,
            rootly: rootlyIcon,
            semaphore: SemaphoreLogo,
            slack: slackIcon,
            sendgrid: sendgridIcon,
            prometheus: prometheusIcon,
            render: renderIcon,
            dockerhub: dockerIcon,
            aws: {
              codeArtifact: awsCodeArtifactIcon,
              cloudwatch: awsCloudwatchIcon,
              ecr: awsEcrIcon,
              lambda: awsLambdaIcon,
              route53: awsRoute53Icon,
              ecs: awsEcsIcon,
              sns: awsSnsIcon,
            },
          };
          const appLogo = nameParts[0] ? appLogoMap[nameParts[0]] : undefined;
          const appIconSrc = typeof appLogo === "string" ? appLogo : nameParts[1] ? appLogo?.[nameParts[1]] : undefined;
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
              className={`ml-3 px-2 py-1 flex items-center gap-2 cursor-grab active:cursor-grabbing ${(() => {
                const subtype = block.componentSubtype || getComponentSubtype(block);
                return subtype === "trigger"
                  ? "hover:bg-sky-100 dark:hover:bg-sky-900/20"
                  : subtype === "flow"
                    ? "hover:bg-purple-100 dark:hover:bg-purple-900/20"
                    : "hover:bg-green-100 dark:hover:bg-green-900/20";
              })()}`}
              size="sm"
            >
              <ItemMedia>
                {appIconSrc ? (
                  <img src={appIconSrc} alt={block.label || block.name} className="size-3.5" />
                ) : (
                  <IconComponent size={14} className="text-gray-500" />
                )}
              </ItemMedia>

              <ItemContent>
                <div className="flex items-center gap-2 w-full min-w-0">
                  <ItemTitle className="text-sm font-normal min-w-0 flex-1 truncate">
                    {block.label || block.name}
                  </ItemTitle>
                  {(() => {
                    const subtype = block.componentSubtype || getComponentSubtype(block);
                    const badgeClass =
                      subtype === "trigger"
                        ? "inline-block text-left px-1.5 py-0.5 text-[11px] font-medium text-sky-600 dark:text-sky-400 rounded whitespace-nowrap flex-shrink-0"
                        : subtype === "flow"
                          ? "inline-block text-left px-1.5 py-0.5 text-[11px] font-medium text-purple-600 dark:text-purple-400 rounded whitespace-nowrap flex-shrink-0"
                          : "inline-block text-left px-1.5 py-0.5 text-[11px] font-medium text-green-600 dark:text-green-400 rounded whitespace-nowrap flex-shrink-0";
                    return (
                      <span className={`${badgeClass} ml-auto`}>
                        {subtype === "trigger" ? "Trigger" : subtype === "flow" ? "Flow" : "Action"}
                      </span>
                    );
                  })()}
                  {block.deprecated && (
                    <span className="px-1.5 py-0.5 text-[11px] font-medium bg-gray-950/5 text-gray-500 rounded whitespace-nowrap flex-shrink-0">
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
