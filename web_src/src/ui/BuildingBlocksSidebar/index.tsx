import type {
  ComponentsEdge,
  ComponentsNode,
  OrganizationsIntegration,
  SuperplaneBlueprintsOutputChannel,
  SuperplaneComponentsOutputChannel,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { isCustomComponentsEnabled } from "@/lib/env";
import { resolveIcon } from "@/lib/utils";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { getBackgroundColorClass } from "@/utils/colors";
import { ChevronRight, GripVerticalIcon, Plug, Plus, Search, Settings2, StickyNote, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toTestId } from "../../utils/testID";
import { getComponentSubtype } from "../buildingBlocks";
import { BuildingBlockPreview } from "./BuildingBlockPreview";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { ComponentBase } from "../componentBase";
import { getHeaderIconSrc, getIntegrationIconSrc } from "../componentSidebar/integrationIcons";
import {
  AiChatSession,
  AiBuilderMessage,
  AiBuilderProposal,
  loadChatConversation,
  loadChatSessions,
  pushAiMessages,
  sendChatPrompt,
} from "./agentChat";
import { AiBuilderChatPanel } from "./AiBuilderChatPanel";

const AI_BUILDER_STORAGE_KEY_PREFIX = "sp:canvas-ai-builder:";

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
  exampleOutput?: Record<string, unknown>;
  exampleData?: Record<string, unknown>;
}

export type BuildingBlockCategory = {
  name: string;
  blocks: BuildingBlock[];
};

export interface BuildingBlocksSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  showAiBuilderTab?: boolean;
  canvasId?: string;
  organizationId?: string;
  canvasNodes?: Array<{
    id: string;
    name?: string;
    label?: string;
    type?: string;
  }>;
  aiCanvas?: {
    name?: string;
    description?: string;
    nodes?: ComponentsNode[];
    edges?: ComponentsEdge[];
  };
  selectedNodeIds?: string[];
  onApplyAiOperations?: (operations: AiCanvasOperation[]) => Promise<void>;
  integrations?: OrganizationsIntegration[];
  canvasZoom?: number;
  disabled?: boolean;
  disabledMessage?: string;
  onBlockClick?: (block: BuildingBlock) => void;
  onAddNote?: () => void;
}

export type AiCanvasOperation =
  | {
      type: "add_node";
      nodeKey?: string;
      blockName: string;
      nodeName?: string;
      configuration?: Record<string, unknown>;
      position?: { x: number; y: number };
      source?: {
        nodeKey?: string;
        nodeId?: string;
        nodeName?: string;
        handleId?: string | null;
      };
    }
  | {
      type: "connect_nodes";
      source: { nodeKey?: string; nodeId?: string; nodeName?: string; handleId?: string | null };
      target: { nodeKey?: string; nodeId?: string; nodeName?: string };
    }
  | {
      type: "disconnect_nodes";
      source: { nodeKey?: string; nodeId?: string; nodeName?: string; handleId?: string | null };
      target: { nodeKey?: string; nodeId?: string; nodeName?: string };
    }
  | {
      type: "update_node_config";
      target: { nodeKey?: string; nodeId?: string; nodeName?: string };
      configuration: Record<string, unknown>;
      nodeName?: string;
    }
  | {
      type: "delete_node";
      target: { nodeKey?: string; nodeId?: string; nodeName?: string };
    };

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  showAiBuilderTab = false,
  canvasId,
  organizationId,
  onApplyAiOperations,
  integrations = [],
  canvasZoom = 1,
  disabled = false,
  disabledMessage,
  onBlockClick,
  onAddNote,
}: BuildingBlocksSidebarProps) {
  const disabledTooltip = disabledMessage || "Finish configuring the selected component first";

  if (!isOpen) {
    return (
      <ClosedBuildingBlocksSidebar
        disabled={disabled}
        disabledTooltip={disabledTooltip}
        onAddNote={onAddNote}
        onToggle={onToggle}
      />
    );
  }

  return (
    <OpenBuildingBlocksSidebar
      onToggle={onToggle}
      blocks={blocks}
      showAiBuilderTab={showAiBuilderTab}
      canvasId={canvasId}
      organizationId={organizationId}
      onApplyAiOperations={onApplyAiOperations}
      integrations={integrations}
      canvasZoom={canvasZoom}
      disabled={disabled}
      disabledTooltip={disabledTooltip}
      onBlockClick={onBlockClick}
    />
  );
}

interface ClosedBuildingBlocksSidebarProps {
  disabled: boolean;
  disabledTooltip: string;
  onAddNote?: () => void;
  onToggle: (open: boolean) => void;
}

function ClosedBuildingBlocksSidebar({
  disabled,
  disabledTooltip,
  onAddNote,
  onToggle,
}: ClosedBuildingBlocksSidebarProps) {
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
      <StickyNote size={16} />
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

interface OpenBuildingBlocksSidebarProps {
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  showAiBuilderTab: boolean;
  canvasId?: string;
  organizationId?: string;
  onApplyAiOperations?: (operations: AiCanvasOperation[]) => Promise<void>;
  integrations: OrganizationsIntegration[];
  canvasZoom: number;
  disabled: boolean;
  disabledTooltip: string;
  onBlockClick?: (block: BuildingBlock) => void;
}

function OpenBuildingBlocksSidebar({
  onToggle,
  blocks,
  showAiBuilderTab,
  canvasId,
  organizationId,
  onApplyAiOperations,
  integrations,
  canvasZoom,
  disabled,
  disabledTooltip,
  onBlockClick,
}: OpenBuildingBlocksSidebarProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const [typeFilter, setTypeFilter] = useState<"all" | "trigger" | "action" | "flow">("all");
  const sidebarRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const aiInputRef = useRef<HTMLTextAreaElement>(null);
  const isDraggingRef = useRef(false);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") {
      return 450;
    }

    const saved = window.localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 450;
  });
  const [isResizing, setIsResizing] = useState(false);
  const [hoveredBlock, setHoveredBlock] = useState<BuildingBlock | null>(null);
  const dragPreviewRef = useRef<HTMLDivElement>(null);
  const [showIntegrationSetupStatus, setShowIntegrationSetupStatus] = useState(true);
  const [showConnectedIntegrationsOnTop, setShowConnectedIntegrationsOnTop] = useState(false);
  const [activeTab, setActiveTab] = useState<"components" | "ai">("components");
  const [aiInput, setAiInput] = useState("");
  const [aiMessages, setAiMessages] = useState<AiBuilderMessage[]>([]);
  const [chatSessions, setChatSessions] = useState<AiChatSession[]>([]);
  const [currentChatId, setCurrentChatId] = useState<string | null>(null);
  const [isLoadingChatSessions, setIsLoadingChatSessions] = useState(false);
  const [isLoadingChatMessages, setIsLoadingChatMessages] = useState(false);
  const [isGeneratingResponse, setIsGeneratingResponse] = useState(false);
  const [isApplyingProposal, setIsApplyingProposal] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);
  const [pendingProposal, setPendingProposal] = useState<AiBuilderProposal | null>(null);
  const applyShortcutHint = useMemo(() => {
    if (typeof navigator === "undefined") {
      return "Ctrl+Enter";
    }

    const isMacPlatform = /Mac|iPhone|iPad|iPod/i.test(`${navigator.platform} ${navigator.userAgent}`);
    return isMacPlatform ? "Cmd+Enter" : "Ctrl+Enter";
  }, []);
  const normalizeIntegrationName = (value?: string) => (value || "").toLowerCase().replace(/[^a-z0-9]/g, "");
  const handleSendPrompt = useCallback(
    async (value?: string) => {
      await sendChatPrompt({
        value,
        aiInput,
        canvasId,
        organizationId,
        currentChatId,
        isGeneratingResponse,
        setChatSessions,
        setCurrentChatId,
        setAiMessages,
        setAiInput,
        setAiError,
        setIsGeneratingResponse,
        setPendingProposal,
        focusInput: () => aiInputRef.current?.focus(),
      });
    },
    [aiInput, canvasId, currentChatId, isGeneratingResponse, organizationId],
  );

  const handleStartNewChatSession = useCallback(() => {
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    requestAnimationFrame(() => {
      aiInputRef.current?.focus();
    });
  }, []);

  const handleSelectChatSession = useCallback((chatId: string) => {
    setCurrentChatId(chatId);
    setPendingProposal(null);
    setAiError(null);
  }, []);

  const handleDiscardProposal = useCallback(() => {
    setPendingProposal(null);
  }, []);

  const formatOperation = useCallback((operation: AiCanvasOperation, proposal?: AiBuilderProposal) => {
    const operationNodeLabels = new Map<string, string>();
    if (proposal) {
      for (const op of proposal.operations) {
        if (op.type === "add_node" && op.nodeKey) {
          operationNodeLabels.set(op.nodeKey, op.nodeName || op.blockName);
        }
      }
    }

    const resolveRefLabel = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
      if (!ref) return "step";
      if (ref.nodeName) return ref.nodeName;
      if (ref.nodeKey && operationNodeLabels.has(ref.nodeKey)) {
        return operationNodeLabels.get(ref.nodeKey) || "step";
      }
      if (ref.nodeId) return ref.nodeId;
      return "step";
    };

    switch (operation.type) {
      case "add_node":
        return `Add node ${operation.nodeName || operation.blockName} (${operation.blockName})`;
      case "connect_nodes":
        return `Connect ${resolveRefLabel(operation.source)} -> ${resolveRefLabel(operation.target)}`;
      case "disconnect_nodes":
        return `Disconnect ${resolveRefLabel(operation.source)} -> ${resolveRefLabel(operation.target)}`;
      case "update_node_config":
        return `Update configuration for ${operation.nodeName || operation.target.nodeName || "node"}`;
      case "delete_node":
        return `Delete node ${resolveRefLabel(operation.target)}`;
      default:
        return "Update canvas";
    }
  }, []);
  const pendingProposalSummaries = useMemo(() => {
    if (!pendingProposal) {
      return [];
    }

    return pendingProposal.operations
      .filter((operation) => operation.type !== "connect_nodes")
      .map((operation) => formatOperation(operation, pendingProposal));
  }, [formatOperation, pendingProposal]);

  const handleApplyProposal = useCallback(async () => {
    if (!pendingProposal) return;

    if (!onApplyAiOperations) {
      setAiError("Canvas apply handlers are not available.");
      return;
    }

    setAiError(null);
    setIsApplyingProposal(true);
    try {
      await onApplyAiOperations(pendingProposal.operations);
      setAiMessages((prev) =>
        pushAiMessages(prev, {
          id: `assistant-${Date.now()}`,
          role: "assistant",
          content: "Applied the proposed changes to the canvas.",
        }),
      );
      setPendingProposal(null);
    } catch (error) {
      setAiError(error instanceof Error ? error.message : "Failed to apply AI proposal.");
    } finally {
      setIsApplyingProposal(false);
    }
  }, [onApplyAiOperations, pendingProposal]);

  useEffect(() => {
    if (activeTab !== "ai" || !pendingProposal || pendingProposal.operations.length === 0) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.isComposing || event.key !== "Enter") {
        return;
      }

      if (!(event.metaKey || event.ctrlKey)) {
        return;
      }

      if (disabled || isApplyingProposal) {
        return;
      }

      event.preventDefault();
      void handleApplyProposal();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [activeTab, disabled, handleApplyProposal, isApplyingProposal, pendingProposal]);

  // Save sidebar width to localStorage whenever it changes
  useEffect(() => {
    localStorage.setItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (!showAiBuilderTab && activeTab === "ai") {
      setActiveTab("components");
    }
  }, [showAiBuilderTab, activeTab]);

  useEffect(() => {
    setActiveTab("components");
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    setAiInput("");
  }, [canvasId]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const keysToRemove: string[] = [];
    for (let index = 0; index < window.localStorage.length; index += 1) {
      const key = window.localStorage.key(index);
      if (key?.startsWith(AI_BUILDER_STORAGE_KEY_PREFIX)) {
        keysToRemove.push(key);
      }
    }

    for (const key of keysToRemove) {
      window.localStorage.removeItem(key);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    if (!canvasId || !organizationId) {
      setChatSessions([]);
      setCurrentChatId(null);
      setAiMessages([]);
      return () => {
        cancelled = true;
      };
    }

    void (async () => {
      setIsLoadingChatSessions(true);
      try {
        const sessions = await loadChatSessions({
          canvasId,
          organizationId,
        });
        if (cancelled) {
          return;
        }

        setChatSessions(sessions);
        setCurrentChatId((previousChatId) => {
          if (previousChatId && sessions.some((session) => session.id === previousChatId)) {
            return previousChatId;
          }

          return null;
        });
      } catch (error) {
        if (!cancelled) {
          console.warn("Failed to load chat sessions:", error);
        }
      } finally {
        if (!cancelled) {
          setIsLoadingChatSessions(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [canvasId, organizationId]);

  useEffect(() => {
    let cancelled = false;

    if (!canvasId || !organizationId || !currentChatId) {
      if (!currentChatId) {
        setAiMessages([]);
        setPendingProposal(null);
      }
      setIsLoadingChatMessages(false);
      return () => {
        cancelled = true;
      };
    }

    void (async () => {
      setIsLoadingChatMessages(true);
      try {
        const messages = await loadChatConversation({
          chatId: currentChatId,
          canvasId,
          organizationId,
        });
        if (cancelled) {
          return;
        }

        setAiMessages(messages);
        setAiError(null);
      } catch (error) {
        if (!cancelled) {
          console.warn("Failed to load chat conversation:", error);
          setAiError(error instanceof Error ? error.message : "Failed to load chat conversation.");
        }
      } finally {
        if (!cancelled) {
          setIsLoadingChatMessages(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [canvasId, currentChatId, organizationId]);

  // Auto-focus search input when sidebar opens
  useEffect(() => {
    if (!searchInputRef.current) {
      return;
    }

    // Small delay to ensure the sidebar is fully rendered
    const timeoutId = window.setTimeout(() => {
      searchInputRef.current?.focus();
    }, 100);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, []);

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

  const sortedCategories = useMemo(() => {
    const categoryOrder: Record<string, number> = {
      Core: 0,
      Memory: 1,
      Bundles: 2,
    };

    const filteredCategories = (blocks || []).filter((category) => {
      if (category.name === "Bundles" && !isCustomComponentsEnabled()) {
        return false;
      }
      return true;
    });

    return [...filteredCategories].sort((a, b) => {
      const aOrder = categoryOrder[a.name] ?? Infinity;
      const bOrder = categoryOrder[b.name] ?? Infinity;

      if (aOrder !== bOrder) {
        return aOrder - bOrder;
      }

      if (showConnectedIntegrationsOnTop && aOrder === Infinity && bOrder === Infinity) {
        const aIntegrationName = a.blocks.find((block) => block.integrationName)?.integrationName;
        const bIntegrationName = b.blocks.find((block) => block.integrationName)?.integrationName;

        const aHasConnectedIntegration = aIntegrationName
          ? integrations.some(
              (integration) =>
                normalizeIntegrationName(integration.spec?.integrationName) ===
                normalizeIntegrationName(aIntegrationName),
            )
          : false;

        const bHasConnectedIntegration = bIntegrationName
          ? integrations.some(
              (integration) =>
                normalizeIntegrationName(integration.spec?.integrationName) ===
                normalizeIntegrationName(bIntegrationName),
            )
          : false;

        if (aHasConnectedIntegration !== bHasConnectedIntegration) {
          return aHasConnectedIntegration ? -1 : 1;
        }
      }

      return a.name.localeCompare(b.name);
    });
  }, [blocks, integrations, showConnectedIntegrationsOnTop]);

  const componentsTabContent = useMemo(
    () => (
      <TabsContent value="components" className="mt-0 flex-1 overflow-y-auto overflow-x-hidden">
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
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon-sm" className="h-8 w-8" aria-label="Sidebar settings">
                <Settings2 className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuCheckboxItem
                checked={showIntegrationSetupStatus}
                onCheckedChange={(checked) => setShowIntegrationSetupStatus(Boolean(checked))}
              >
                Show integration setup status
              </DropdownMenuCheckboxItem>
              <DropdownMenuCheckboxItem
                checked={showConnectedIntegrationsOnTop}
                onCheckedChange={(checked) => setShowConnectedIntegrationsOnTop(Boolean(checked))}
              >
                Connected integrations on top
              </DropdownMenuCheckboxItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        <div className="gap-2 py-6">
          {sortedCategories.map((category) => (
            <CategorySection
              key={category.name}
              category={category}
              integrations={integrations}
              showIntegrationSetupStatus={showIntegrationSetupStatus}
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
      </TabsContent>
    ),
    [
      canvasZoom,
      disabled,
      disabledTooltip,
      integrations,
      onBlockClick,
      searchTerm,
      showConnectedIntegrationsOnTop,
      showIntegrationSetupStatus,
      sortedCategories,
      typeFilter,
    ],
  );

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-21 overflow-y-auto overflow-x-hidden bg-white"
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

      {!showAiBuilderTab && (
        <div className="flex items-center justify-between gap-3 px-5 py-4 relative">
          <div className="flex flex-col items-start gap-3 w-full">
            <div className="flex justify-between gap-3 w-full">
              <div className="flex flex-col gap-0.5">
                <h2 className="text-base font-medium">Add Component</h2>
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
      )}

      <Tabs
        value={showAiBuilderTab ? activeTab : "components"}
        onValueChange={(value) => setActiveTab(value as "components" | "ai")}
        className={`flex ${showAiBuilderTab ? "h-full" : "h-[calc(100%-82px)]"} flex-col`}
      >
        {showAiBuilderTab && (
          <div className="px-4 pt-3 pb-3 flex items-center gap-1.5 relative">
            <TabsList className="grid h-8 w-auto grid-cols-2 gap-0.5 bg-transparent p-0">
              <TabsTrigger
                value="components"
                className="h-7 rounded-sm px-2 text-xs text-muted-foreground shadow-none data-[state=active]:bg-muted data-[state=active]:text-foreground data-[state=active]:shadow-none"
              >
                Components
              </TabsTrigger>
              <TabsTrigger
                value="ai"
                className="h-7 gap-1 rounded-sm px-2 text-xs text-muted-foreground shadow-none data-[state=active]:bg-muted data-[state=active]:text-foreground data-[state=active]:shadow-none"
              >
                <span>AI Builder</span>
                {pendingProposal ? <span className="h-2 w-2 rounded-full bg-blue-500" /> : null}
              </TabsTrigger>
            </TabsList>
            <div
              onClick={() => onToggle(false)}
              className="absolute top-4 right-4 w-6 h-6 hover:bg-slate-950/5 rounded flex items-center justify-center cursor-pointer leading-none"
            >
              <X size={16} />
            </div>
          </div>
        )}
        {(!showAiBuilderTab || activeTab === "components") && componentsTabContent}

        {showAiBuilderTab && (
          <AiBuilderChatPanel
            chatSessions={chatSessions}
            currentChatId={currentChatId}
            isLoadingChatSessions={isLoadingChatSessions}
            isLoadingChatMessages={isLoadingChatMessages}
            aiMessages={aiMessages}
            isGeneratingResponse={isGeneratingResponse}
            pendingProposal={pendingProposal}
            pendingProposalSummaries={pendingProposalSummaries}
            applyShortcutHint={applyShortcutHint}
            onApplyProposal={() => void handleApplyProposal()}
            onDiscardProposal={handleDiscardProposal}
            isApplyingProposal={isApplyingProposal}
            aiError={aiError}
            disabled={disabled}
            canvasId={canvasId}
            aiInput={aiInput}
            onAiInputChange={setAiInput}
            onSelectChat={handleSelectChatSession}
            onStartNewSession={handleStartNewChatSession}
            onSendPrompt={() => void handleSendPrompt()}
            aiInputRef={aiInputRef}
          />
        )}
      </Tabs>

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
  isLive: boolean;
  canvasZoom: number;
  isDraggingRef: React.RefObject<boolean>;
  setHoveredBlock: (block: BuildingBlock | null) => void;
  dragPreviewRef: React.RefObject<HTMLDivElement | null>;
  onBlockClick?: (block: BuildingBlock) => void;
}

function BlockItem({
  block,
  isLive,
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
        draggable={isLive}
        onClick={() => {
          if (isLive && onBlockClick) onBlockClick(block);
        }}
        onMouseEnter={() => {
          if (isLive) setHoveredBlock(block);
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
          setupDragPreview(e, dragPreviewRef, canvasZoom);
        }}
        onDragEnd={() => {
          isDraggingRef.current = false;
          setHoveredBlock(null);
        }}
        aria-disabled={!isLive}
        title={isLive ? undefined : "Coming soon"}
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

interface CategorySectionProps {
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

function CategorySection({
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

  const subtypeOrder: Record<"trigger" | "action" | "flow", number> = {
    trigger: 0,
    action: 1,
    flow: 2,
  };

  const sortedBlocks = [...allBlocks].sort((a, b) => {
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

  // Get integration name from first block if available, or match category name
  const firstBlock = allBlocks[0];
  const integrationName = firstBlock?.integrationName || category.name.toLowerCase();
  const categoryIconSrc = integrationName === "smtp" ? undefined : getIntegrationIconSrc(integrationName);

  // Mirror org/integrations colors: ready=green, pending=amber, error=red, default=gray.
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

  // Determine icon for special categories (Core, Bundles, SMTP use Lucide SVG; others use img when categoryIconSrc)
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

  const isCoreCategory = category.name === "Core";
  const hasSearchTerm = query.length > 0;
  // Expand if it's Core category (default) or if there's a search term (show results)
  const shouldBeOpen = isCoreCategory || hasSearchTerm;

  return (
    <details className="flex-1 px-5 mb-5 group" open={shouldBeOpen}>
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

      <ItemGroup>
        {sortedBlocks.map((block) => (
          <BlockItem
            key={`${block.type}-${block.name}`}
            block={block}
            isLive={!!block.isLive}
            canvasZoom={canvasZoom}
            isDraggingRef={isDraggingRef}
            setHoveredBlock={setHoveredBlock}
            dragPreviewRef={dragPreviewRef}
            onBlockClick={onBlockClick}
          />
        ))}
      </ItemGroup>
    </details>
  );
}
