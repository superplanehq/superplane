import type {
  OrganizationsIntegration,
  SuperplaneBlueprintsOutputChannel,
  SuperplaneComponentsOutputChannel,
} from "@/api-client";
import { canvasesSendAiMessage } from "@/api-client/sdk.gen";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Item, ItemContent, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { resolveIcon } from "@/lib/utils";
import { isCustomComponentsEnabled } from "@/lib/env";
import { getBackgroundColorClass } from "@/utils/colors";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { getComponentSubtype } from "../buildingBlocks";
import {
  ChevronRight,
  GripVerticalIcon,
  Plug,
  Plus,
  Search,
  SendHorizontal,
  Settings2,
  StickyNote,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toTestId } from "../../utils/testID";
import { COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY } from "../CanvasPage";
import { ComponentBase } from "../componentBase";
import { loadAiBuilderState, saveAiBuilderState } from "./aiBuilderStorage";
import circleciIcon from "@/assets/icons/integrations/circleci.svg";
import cloudflareIcon from "@/assets/icons/integrations/cloudflare.svg";
import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import daytonaIcon from "@/assets/icons/integrations/daytona.svg";
import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import digitaloceanIcon from "@/assets/icons/integrations/digitalocean.svg";
import discordIcon from "@/assets/icons/integrations/discord.svg";
import telegramIcon from "@/assets/icons/integrations/telegram.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import gitlabIcon from "@/assets/icons/integrations/gitlab.svg";
import jiraIcon from "@/assets/icons/integrations/jira.svg";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";
import cursorIcon from "@/assets/icons/integrations/cursor.svg";
import pagerDutyIcon from "@/assets/icons/integrations/pagerduty.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import awsIcon from "@/assets/icons/integrations/aws.svg";
import awsLambdaIcon from "@/assets/icons/integrations/aws.lambda.svg";
import awsRoute53Icon from "@/assets/icons/integrations/aws.route53.svg";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";
import awsEcrIcon from "@/assets/icons/integrations/aws.ecr.svg";
import awsEcsIcon from "@/assets/icons/integrations/aws.ecs.svg";
import awsCodeArtifactIcon from "@/assets/icons/integrations/aws.codeartifact.svg";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import awsCodePipelineIcon from "@/assets/icons/integrations/aws.codepipeline.svg";
import awsSnsIcon from "@/assets/icons/integrations/aws.sns.svg";
import rootlyIcon from "@/assets/icons/integrations/rootly.svg";
import incidentIcon from "@/assets/icons/integrations/incident.svg";
import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import sendgridIcon from "@/assets/icons/integrations/sendgrid.svg";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";
import renderIcon from "@/assets/icons/integrations/render.svg";
import dockerIcon from "@/assets/icons/integrations/docker.svg";
import awsSqsIcon from "@/assets/icons/integrations/aws.sqs.svg";
import hetznerIcon from "@/assets/icons/integrations/hetzner.svg";
import jfrogArtifactoryIcon from "@/assets/icons/integrations/jfrog-artifactory.svg";
import harnessIcon from "@/assets/icons/integrations/harness.svg";
import servicenowIcon from "@/assets/icons/integrations/servicenow.svg";
import statuspageIcon from "@/assets/icons/integrations/statuspage.svg";

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
  showAiBuilderTab?: boolean;
  canvasId?: string;
  canvasNodes?: Array<{
    id: string;
    name?: string;
    label?: string;
    type?: string;
  }>;
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
      type: "update_node_config";
      target: { nodeKey?: string; nodeId?: string; nodeName?: string };
      configuration: Record<string, unknown>;
      nodeName?: string;
    }
  | {
      type: "delete_node";
      target: { nodeKey?: string; nodeId?: string; nodeName?: string };
    };

type AiBuilderMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
};

type AiBuilderProposal = {
  id: string;
  summary: string;
  operations: AiCanvasOperation[];
};

const AI_HISTORY_RECENT_TURNS = 8;
const AI_HISTORY_OLDER_TURNS = 6;
const AI_HISTORY_MAX_MESSAGE_CHARS = 320;
const AI_MAX_STORED_MESSAGES = 50;

function compactMessageContent(content: string): string {
  const normalized = content.replace(/\s+/g, " ").trim();
  if (normalized.length <= AI_HISTORY_MAX_MESSAGE_CHARS) {
    return normalized;
  }

  return `${normalized.slice(0, AI_HISTORY_MAX_MESSAGE_CHARS)}...`;
}

function formatConversationTurns(messages: AiBuilderMessage[]): string[] {
  return messages
    .map((message) => `${message.role}: ${compactMessageContent(message.content)}`)
    .filter((line) => line.length > 0);
}

function buildPromptWithConversationContext(messages: AiBuilderMessage[], prompt: string): string {
  const turns = formatConversationTurns(messages);
  if (turns.length === 0) {
    return prompt;
  }

  const recentTurns = turns.slice(-AI_HISTORY_RECENT_TURNS);
  const olderTurns = turns.slice(0, -AI_HISTORY_RECENT_TURNS).slice(-AI_HISTORY_OLDER_TURNS);
  const contextSections = [
    "Conversation context (use this for continuity and intent resolution):",
    ...(olderTurns.length > 0 ? [`Earlier turns summary:\n${olderTurns.join("\n")}`] : []),
    `Recent turns:\n${recentTurns.join("\n")}`,
    `Current user request:\n${prompt}`,
  ];

  return contextSections.join("\n\n");
}

function pushAiMessages(previous: AiBuilderMessage[], next: AiBuilderMessage | AiBuilderMessage[]): AiBuilderMessage[] {
  const nextMessages = Array.isArray(next) ? next : [next];
  const merged = [...previous, ...nextMessages];
  if (merged.length <= AI_MAX_STORED_MESSAGES) {
    return merged;
  }

  return merged.slice(-AI_MAX_STORED_MESSAGES);
}

export function BuildingBlocksSidebar({
  isOpen,
  onToggle,
  blocks,
  showAiBuilderTab = false,
  canvasId,
  canvasNodes = [],
  onApplyAiOperations,
  integrations = [],
  canvasZoom = 1,
  disabled = false,
  disabledMessage,
  onBlockClick,
  onAddNote,
}: BuildingBlocksSidebarProps) {
  const disabledTooltip = disabledMessage || "Finish configuring the selected component first";
  const persistedAiState = loadAiBuilderState<AiCanvasOperation>(canvasId);

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
  const aiInputRef = useRef<HTMLInputElement>(null);
  const isDraggingRef = useRef(false);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    const saved = localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 450;
  });
  const [isResizing, setIsResizing] = useState(false);
  const [hoveredBlock, setHoveredBlock] = useState<BuildingBlock | null>(null);
  const dragPreviewRef = useRef<HTMLDivElement>(null);
  const [showIntegrationSetupStatus, setShowIntegrationSetupStatus] = useState(true);
  const [showConnectedIntegrationsOnTop, setShowConnectedIntegrationsOnTop] = useState(false);
  const [activeTab, setActiveTab] = useState<"components" | "ai">(persistedAiState?.activeTab || "components");
  const [aiInput, setAiInput] = useState("");
  const [aiMessages, setAiMessages] = useState<AiBuilderMessage[]>(persistedAiState?.messages || []);
  const [isGeneratingResponse, setIsGeneratingResponse] = useState(false);
  const [isApplyingProposal, setIsApplyingProposal] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);
  const [pendingProposal, setPendingProposal] = useState<AiBuilderProposal | null>(
    persistedAiState?.pendingProposal || null,
  );
  const aiMessagesContainerRef = useRef<HTMLDivElement>(null);

  const normalizeIntegrationName = (value?: string) => (value || "").toLowerCase().replace(/[^a-z0-9]/g, "");
  const handleSendPrompt = useCallback(
    async (value?: string) => {
      const nextPrompt = (value ?? aiInput).trim();
      if (!nextPrompt || isGeneratingResponse || !canvasId) {
        return;
      }

      if (nextPrompt.toLowerCase() === "/clear") {
        setAiMessages([]);
        setPendingProposal(null);
        setAiError(null);
        setAiInput("");
        requestAnimationFrame(() => {
          aiInputRef.current?.focus();
        });
        return;
      }

      const contextualPrompt = buildPromptWithConversationContext(aiMessages, nextPrompt);

      const userMessage: AiBuilderMessage = {
        id: `user-${Date.now()}`,
        role: "user",
        content: nextPrompt,
      };
      setAiMessages((prev) => pushAiMessages(prev, userMessage));
      setAiInput("");
      requestAnimationFrame(() => {
        aiInputRef.current?.focus();
      });
      setAiError(null);
      setIsGeneratingResponse(true);

      try {
        const availableBlocks = (blocks || []).flatMap((category) =>
          category.blocks
            .filter((block) => block.isLive && !block.deprecated)
            .map((block) => ({
              name: block.name,
              label: block.label || block.name,
              type: block.type,
            })),
        );

        const apiResponse = await canvasesSendAiMessage(
          withOrganizationHeader({
            path: { canvasId },
            body: {
              prompt: contextualPrompt,
              canvasContext: {
                nodes: canvasNodes,
                availableBlocks,
              },
            },
          }),
        );

        const payload = apiResponse.data as { assistantMessage?: string; operations?: AiCanvasOperation[] } | undefined;
        const proposal: AiBuilderProposal = {
          id: `proposal-${Date.now()}`,
          summary: payload?.assistantMessage || "I prepared a draft change set you can review and apply.",
          operations: payload?.operations || [],
        };

        const assistantMessage: AiBuilderMessage = {
          id: `assistant-${Date.now()}`,
          role: "assistant",
          content: proposal.summary,
        };
        setAiMessages((prev) => pushAiMessages(prev, assistantMessage));
        if (proposal.operations.length > 0) {
          setPendingProposal(proposal);
        } else {
          setPendingProposal(null);
        }
      } catch (error) {
        const fallbackMessage = "I couldn't generate changes right now. Please try again.";
        setAiError(error instanceof Error ? error.message : fallbackMessage);
        setAiMessages((prev) =>
          pushAiMessages(prev, {
            id: `assistant-${Date.now()}`,
            role: "assistant",
            content: fallbackMessage,
          }),
        );
      } finally {
        setIsGeneratingResponse(false);
      }
    },
    [aiInput, aiMessages, blocks, canvasId, canvasNodes, isGeneratingResponse],
  );

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
      case "update_node_config":
        return `Update configuration for ${operation.nodeName || operation.target.nodeName || "node"}`;
      case "delete_node":
        return `Delete node ${resolveRefLabel(operation.target)}`;
      default:
        return "Update canvas";
    }
  }, []);

  const extractAssistantOptions = useCallback((content: string): string[] => {
    const optionSet = new Set<string>();

    const lines = content
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);

    for (const line of lines) {
      const bulletMatch = line.match(/^[-*]\s+(.+)$/);
      const numberedMatch = line.match(/^\d+[.)]\s+(.+)$/);
      const optionText = bulletMatch?.[1] || numberedMatch?.[1];
      if (!optionText) continue;

      const normalized = optionText.replace(/\s+/g, " ").trim();
      if (normalized.length < 2 || normalized.length > 140) continue;
      optionSet.add(normalized.replace(/[.;,\s]+$/, ""));
    }

    if (optionSet.size === 0) {
      const codeMatches = [...content.matchAll(/`([^`]+)`/g)];
      for (const match of codeMatches) {
        const value = (match[1] || "").trim();
        if (!value || value.length > 80) continue;
        if (value.toLowerCase() === "etc.") continue;
        optionSet.add(value);
      }
    }

    return Array.from(optionSet).slice(0, 8);
  }, []);

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
    const nextPersistedState = loadAiBuilderState<AiCanvasOperation>(canvasId);
    setActiveTab(nextPersistedState?.activeTab || "components");
    setAiMessages(nextPersistedState?.messages || []);
    setPendingProposal(nextPersistedState?.pendingProposal || null);
    setAiError(null);
    setAiInput("");
  }, [canvasId]);

  useEffect(() => {
    saveAiBuilderState<AiCanvasOperation>(canvasId, {
      activeTab,
      messages: aiMessages,
      pendingProposal,
    });
  }, [activeTab, aiMessages, canvasId, pendingProposal]);

  useEffect(() => {
    if (activeTab !== "ai") {
      return;
    }

    const container = aiMessagesContainerRef.current;
    if (!container) {
      return;
    }

    container.scrollTop = container.scrollHeight;
  }, [activeTab, aiMessages, pendingProposal, isGeneratingResponse, aiError]);

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

  const sortedCategories = useMemo(() => {
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
          <TabsContent value="ai" className="mt-0 flex-1 overflow-hidden px-5 pb-5">
            <div className="h-full rounded-md border border-border bg-slate-50/30 flex flex-col">
              <div ref={aiMessagesContainerRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
                {aiMessages.length === 0 ? (
                  <div className="text-sm text-gray-600">
                    <div className="flex items-start gap-2">
                      <p>Describe your flow and I will propose changes.</p>
                    </div>
                  </div>
                ) : (
                  <>
                    {aiMessages.map((message) => (
                      <div key={message.id} className={message.role === "user" ? "ml-6" : "mr-6"}>
                        <div
                          className={
                            message.role === "user"
                              ? "rounded-md bg-blue-600 text-white px-3 py-2 text-sm"
                              : "rounded-md bg-white border border-border px-3 py-2 text-sm text-gray-800"
                          }
                        >
                          {message.content}
                        </div>
                        {message.role === "assistant" && !pendingProposal
                          ? (() => {
                              const options = extractAssistantOptions(message.content);
                              if (options.length === 0) return null;
                              return (
                                <div className="mt-2 flex flex-wrap gap-2">
                                  {options.map((option) => (
                                    <Button
                                      key={`${message.id}-${option}`}
                                      type="button"
                                      size="sm"
                                      variant="outline"
                                      className="h-7 text-xs"
                                      onClick={() => handleSendPrompt(option)}
                                      disabled={disabled || isGeneratingResponse || !canvasId}
                                    >
                                      {option}
                                    </Button>
                                  ))}
                                </div>
                              );
                            })()
                          : null}
                      </div>
                    ))}
                    {isGeneratingResponse ? (
                      <div className="sp-ai-thinking text-xs text-gray-500 px-1 py-1 rounded-sm">
                        Planing next steps...
                      </div>
                    ) : null}
                  </>
                )}

                {pendingProposal && (
                  <div className="rounded-md border border-blue-200 bg-blue-50 px-3 py-3 space-y-2">
                    <ul className="text-sm text-blue-900 list-disc pl-5 space-y-1">
                      {pendingProposal.operations
                        .filter((operation) => operation.type !== "connect_nodes")
                        .map((operation) => (
                          <li key={`${pendingProposal.id}-${JSON.stringify(operation)}`}>
                            {formatOperation(operation, pendingProposal)}
                          </li>
                        ))}
                    </ul>
                    <div className="flex items-center gap-2 pt-1">
                      <Button
                        size="sm"
                        onClick={handleApplyProposal}
                        disabled={disabled || isApplyingProposal || pendingProposal.operations.length === 0}
                      >
                        Apply changes
                      </Button>
                      <Button size="sm" variant="outline" onClick={handleDiscardProposal} disabled={isApplyingProposal}>
                        Discard
                      </Button>
                    </div>
                    {aiError ? <p className="text-xs text-red-700">{aiError}</p> : null}
                  </div>
                )}

                {!pendingProposal && aiError ? <p className="text-xs text-red-700">{aiError}</p> : null}
              </div>

              <div className="border-t border-border px-4 py-3">
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    handleSendPrompt();
                  }}
                  className="flex items-center gap-2"
                >
                  <Input
                    ref={aiInputRef}
                    value={aiInput}
                    onChange={(e) => setAiInput(e.target.value)}
                    placeholder="Describe your canvas changes..."
                    disabled={disabled || !canvasId}
                  />
                  <Button
                    type="submit"
                    size="icon-sm"
                    disabled={disabled || isGeneratingResponse || !canvasId || !aiInput.trim()}
                    aria-label="Send prompt"
                  >
                    <SendHorizontal size={14} />
                  </Button>
                </form>
              </div>
            </div>
          </TabsContent>
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

  // Determine category icon
  const appLogoMap: Record<string, string | Record<string, string>> = {
    bitbucket: bitbucketIcon,
    circleci: circleciIcon,
    cloudflare: cloudflareIcon,
    dash0: dash0Icon,
    datadog: datadogIcon,
    daytona: daytonaIcon,
    digitalocean: digitaloceanIcon,
    discord: discordIcon,
    github: githubIcon,
    gitlab: gitlabIcon,
    hetzner: hetznerIcon,
    jfrogArtifactory: jfrogArtifactoryIcon,
    grafana: grafanaIcon,
    jira: jiraIcon,
    openai: openAiIcon,
    "open-ai": openAiIcon,
    claude: claudeIcon,
    cursor: cursorIcon,
    pagerduty: pagerDutyIcon,
    rootly: rootlyIcon,
    incident: incidentIcon,
    semaphore: SemaphoreLogo,
    slack: slackIcon,
    telegram: telegramIcon,
    sendgrid: sendgridIcon,
    prometheus: prometheusIcon,
    render: renderIcon,
    dockerhub: dockerIcon,
    harness: harnessIcon,
    servicenow: servicenowIcon,
    statuspage: statuspageIcon,
    aws: {
      ec2: awsEc2Icon,
      codeArtifact: awsIcon,
      cloudwatch: awsCloudwatchIcon,
      lambda: awsLambdaIcon,
      ecr: awsEcrIcon,
      sqs: awsSqsIcon,
      route53: awsRoute53Icon,
      ecs: awsEcsIcon,
      sns: awsSnsIcon,
    },
    gcp: gcpIcon,
  };

  // Get integration name from first block if available, or match category name
  const firstBlock = allBlocks[0];
  const integrationName = firstBlock?.integrationName || category.name.toLowerCase();
  const appLogo = appLogoMap[integrationName];
  const categoryIconSrc = typeof appLogo === "string" ? appLogo : integrationName === "aws" ? awsIcon : undefined;

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
    category.name === "Core"
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
        {sortedBlocks.map((block) => {
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
            digitalocean: digitaloceanIcon,
            discord: discordIcon,
            github: githubIcon,
            gitlab: gitlabIcon,
            hetzner: hetznerIcon,
            jfrogArtifactory: jfrogArtifactoryIcon,
            grafana: grafanaIcon,
            openai: openAiIcon,
            "open-ai": openAiIcon,
            claude: claudeIcon,
            cursor: cursorIcon,
            pagerduty: pagerDutyIcon,
            rootly: rootlyIcon,
            incident: incidentIcon,
            semaphore: SemaphoreLogo,
            slack: slackIcon,
            telegram: telegramIcon,
            sendgrid: sendgridIcon,
            prometheus: prometheusIcon,
            render: renderIcon,
            dockerhub: dockerIcon,
            harness: harnessIcon,
            servicenow: servicenowIcon,
            statuspage: statuspageIcon,
            aws: {
              codeArtifact: awsCodeArtifactIcon,
              codepipeline: awsCodePipelineIcon,
              cloudwatch: awsCloudwatchIcon,
              ecr: awsEcrIcon,
              ec2: awsEc2Icon,
              lambda: awsLambdaIcon,
              sqs: awsSqsIcon,
              route53: awsRoute53Icon,
              ecs: awsEcsIcon,
              sns: awsSnsIcon,
            },
            gcp: gcpIcon,
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
