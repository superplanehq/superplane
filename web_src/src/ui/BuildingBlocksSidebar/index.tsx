import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getBackgroundColorClass } from "@/lib/colors";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Search, Settings2, X } from "lucide-react";
import { type Dispatch, type SetStateAction, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { ComponentBase } from "../componentBase";
import { CategorySection } from "./CategorySection";
import type { BuildingBlock, BuildingBlockCategory } from "./types";

export type { AgentContext, AgentMode } from "@/components/AgentSidebar/agentChat";
export type { BuildingBlock, BuildingBlockCategory } from "./types";

export interface BuildingBlocksSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  integrations?: OrganizationsIntegration[];
  canvasZoom?: number;
  disabled?: boolean;
  disabledMessage?: string;
  onBlockClick?: (block: BuildingBlock) => void;
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  integrations = [],
  canvasZoom = 1,
  disabled = false,
  disabledMessage,
  onBlockClick,
}: BuildingBlocksSidebarProps) {
  const disabledTooltip = disabledMessage || "Finish configuring the selected component first";
  const [activeTab, setActiveTab] = useState<"components" | "ai">("components");
  const [currentChatId, setCurrentChatId] = useState<string | null>(null);

  useEffect(() => {
    setActiveTab("components");
    setCurrentChatId(null);
  }, [canvasId]);

  if (!isOpen) {
    return null;
  }

  return (
    <OpenBuildingBlocksSidebar
      onToggle={onToggle}
      blocks={blocks}
      integrations={integrations}
      canvasZoom={canvasZoom}
      disabled={disabled}
      disabledTooltip={disabledTooltip}
      onBlockClick={onBlockClick}
      activeTab={activeTab}
      setActiveTab={setActiveTab}
      currentChatId={currentChatId}
      setCurrentChatId={setCurrentChatId}
    />
  );
}

interface OpenBuildingBlocksSidebarProps {
  onToggle: (open: boolean) => void;
  blocks: BuildingBlockCategory[];
  integrations: OrganizationsIntegration[];
  canvasZoom: number;
  disabled: boolean;
  disabledTooltip: string;
  onBlockClick?: (block: BuildingBlock) => void;
  activeTab: "components" | "ai";
  setActiveTab: Dispatch<SetStateAction<"components" | "ai">>;
  currentChatId: string | null;
  setCurrentChatId: Dispatch<SetStateAction<string | null>>;
}

function OpenBuildingBlocksSidebar({
  onToggle,
  blocks,
  integrations,
  canvasZoom,
  disabled,
  disabledTooltip,
  onBlockClick,
  activeTab,
  setActiveTab,
  currentChatId,
  setCurrentChatId,
}: OpenBuildingBlocksSidebarProps) {
  const [searchTerm, setSearchTerm] = useState("");
  const [typeFilter, setTypeFilter] = useState<"all" | "trigger" | "component">("all");
  const sidebarRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
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
  const [aiInput, setAiInput] = useState("");
  const [aiMessages, setAiMessages] = useState<AiBuilderMessage[]>([]);
  const [chatSessions, setChatSessions] = useState<AiChatSession[]>([]);
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
        agentContext,
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
    [agentContext, aiInput, canvasId, currentChatId, isGeneratingResponse, organizationId, setCurrentChatId],
  );

  const handleStartNewChatSession = useCallback(() => {
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    requestAnimationFrame(() => {
      aiInputRef.current?.focus();
    });
  }, [setCurrentChatId]);

  const handleSelectChatSession = useCallback(
    (chatId: string) => {
      setCurrentChatId(chatId);
      setPendingProposal(null);
      setAiError(null);
    },
    [setCurrentChatId],
  );

  const handleDiscardProposal = useCallback(() => {
    setPendingProposal(null);
  }, []);

  const formatOperation = useCallback((change: CanvasChangesetChange) => {
    const getNodeId = (nodeId?: string) => nodeId || "node";

    switch (change.type) {
      case "ADD_NODE":
        return `Add node ${getNodeId(change.node?.id)} (${change.node?.block || "unknown"})`;
      case "UPDATE_NODE":
        return `Update node ${getNodeId(change.node?.id)}`;
      case "DELETE_NODE":
        return `Delete node ${getNodeId(change.node?.id)}`;
      case "ADD_EDGE":
        return `Connect ${getNodeId(change.edge?.sourceId)} -> ${getNodeId(change.edge?.targetId)}`;
      case "DELETE_EDGE":
        return `Disconnect ${getNodeId(change.edge?.sourceId)} -> ${getNodeId(change.edge?.targetId)}`;
      default:
        return "Update canvas";
    }
  }, []);

  const pendingProposalSummaries = useMemo(() => {
    if (!pendingProposal) {
      return [];
    }

    return (pendingProposal.changeset.changes || []).map((change) => formatOperation(change));
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
      await onApplyAiOperations(pendingProposal.changeset.changes || []);
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
    localStorage.setItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (!agentContext.enabled && activeTab === "ai") {
      setActiveTab("components");
    }
  }, [agentContext.enabled, activeTab, setActiveTab]);

  useEffect(() => {
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
  }, [canvasId, organizationId, setCurrentChatId]);

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

    const timeoutId = window.setTimeout(() => {
      searchInputRef.current?.focus();
    }, 100);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, []);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!isResizing) return;

      const newWidth = window.innerWidth - e.clientX;
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

  const normalizeIntegrationName = (value?: string) => (value || "").toLowerCase().replace(/[^a-z0-9]/g, "");

  const sortedCategories = useMemo(() => {
    const categoryOrder: Record<string, number> = {
      Core: 0,
      Memory: 1,
    };

    return [...blocks].sort((a, b) => {
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

  return (
    <div
      ref={sidebarRef}
      className="border-l-1 border-border absolute right-0 top-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="building-blocks-sidebar"
    >
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

      <div className="flex items-center justify-between gap-3 px-5 py-4 shrink-0 border-b border-border/60">
        <div className="flex flex-col items-start gap-0.5 min-w-0">
          <h2 className="text-base font-medium">Add Component</h2>
        </div>
        <button
          type="button"
          onClick={() => onToggle(false)}
          data-testid="close-sidebar-button"
          className="shrink-0 z-40 w-8 h-8 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer leading-none border border-transparent text-muted-foreground"
          aria-label="Close sidebar"
        >
          <X size={16} />
        </button>
      </div>

      <div className="flex flex-1 flex-col min-h-0 overflow-y-auto overflow-x-hidden">
        <div className="flex items-center gap-2 px-5 pt-3 shrink-0">
          <div className="flex-1 relative min-w-0">
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
            <SelectTrigger size="sm" className="w-[120px] shrink-0">
              <SelectValue placeholder="All Types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Types</SelectItem>
              <SelectItem value="trigger">Trigger</SelectItem>
              <SelectItem value="component">Action</SelectItem>
            </SelectContent>
          </Select>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon-sm" className="h-8 w-8 shrink-0" aria-label="Sidebar settings">
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

        <div className="relative flex-1 min-h-0 gap-2 py-6">
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
      </div>

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
