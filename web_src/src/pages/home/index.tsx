import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Box, GitBranch, MoreVertical, Palette, Pencil, Plus, Search, Trash2 } from "lucide-react";
import { useState, type MouseEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { CreateCanvasModal } from "../../components/CreateCanvasModal";
import { CreateCustomComponentModal } from "../../components/CreateCustomComponentModal";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "../../components/Dialog/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../../ui/dropdownMenu";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/AccountContext";
import { useBlueprints, useDeleteBlueprint } from "../../hooks/useBlueprintData";
import { useDeleteWorkflow, useWorkflows, workflowKeys } from "../../hooks/useWorkflowData";
import { resolveIcon } from "../../lib/utils";
import { isCustomComponentsEnabled } from "../../lib/env";
import { showErrorToast, showSuccessToast } from "../../utils/toast";

import { Button } from "@/components/ui/button";
import { useCreateCanvasModalState } from "./useCreateCanvasModalState";
import { useCreateCustomComponentModalState } from "./useCreateCustomComponentModalState";
import type { ComponentsEdge, ComponentsNode } from "@/api-client";

type TabType = "canvases" | "custom-components";
interface BlueprintCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "blueprint";
  createdBy?: { id?: string; name?: string };
}

interface WorkflowCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "workflow";
  createdBy?: { id?: string; name?: string };
  nodes?: ComponentsNode[];
  edges?: ComponentsEdge[];
}

const HomePage = () => {
  usePageTitle(["Home"]);

  const [searchQuery, setSearchQuery] = useState("");
  const [activeTab, setActiveTab] = useState<TabType>("canvases");

  const canvasModalState = useCreateCanvasModalState();
  const customComponentModalState = useCreateCustomComponentModalState();

  const { organizationId } = useParams<{ organizationId: string }>();
  const { account } = useAccount();

  const blueprintsQuery = useBlueprints(organizationId || "");
  const {
    data: blueprintsData = [],
    isLoading: blueprintsLoading,
    error: blueprintApiError,
  } = isCustomComponentsEnabled() ? blueprintsQuery : { data: [], isLoading: false, error: null };

  const {
    data: workflowsData = [],
    isLoading: workflowsLoading,
    error: workflowApiError,
  } = useWorkflows(organizationId || "");

  const blueprintError = blueprintApiError ? "Failed to fetch Bundles. Please try again later." : null;
  const workflowError = workflowApiError ? "Failed to fetch workflows. Please try again later." : null;

  const formatDate = (value?: string) => {
    if (!value) return "Unknown";
    return new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  };

  const blueprints: BlueprintCardData[] = (blueprintsData || []).map((blueprint: any) => ({
    id: blueprint.id!,
    name: blueprint.name!,
    description: blueprint.description,
    createdAt: formatDate(blueprint.createdAt),
    type: "blueprint" as const,
    createdBy: blueprint.createdBy,
  }));

  const workflows: WorkflowCardData[] = (workflowsData || []).map((workflow: any) => ({
    id: workflow.metadata?.id!,
    name: workflow.metadata?.name!,
    description: workflow.metadata?.description,
    createdAt: formatDate(workflow.metadata?.createdAt),
    type: "workflow" as const,
    createdBy: workflow.metadata?.createdBy,
    nodes: workflow.spec?.nodes || [],
    edges: workflow.spec?.edges || [],
  }));

  const filteredBlueprints = blueprints.filter((blueprint) => {
    const matchesSearch =
      blueprint.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      blueprint.description?.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesSearch;
  });

  const filteredWorkflows = workflows.filter((workflow) => {
    const matchesSearch =
      workflow.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      workflow.description?.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesSearch;
  });

  const isLoading =
    (activeTab === "custom-components" && blueprintsLoading) || (activeTab === "canvases" && workflowsLoading);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-40">
        <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600"></div>
        <p className="ml-3 text-gray-500">Loading...</p>
      </div>
    );
  }

  if (!account || !organizationId) {
    return (
      <div className="text-center py-8">
        <p className="text-gray-500">Unable to load user information</p>
      </div>
    );
  }

  const error = activeTab === "custom-components" ? blueprintError : workflowError;

  const onNewClick = () => {
    if (activeTab === "custom-components" && isCustomComponentsEnabled()) {
      customComponentModalState.onOpen();
    } else {
      canvasModalState.onOpen();
    }
  };
  const showTabs = false;

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <header className="bg-white border-b border-border px-4 h-12 flex items-center">
        <OrganizationMenuButton organizationId={organizationId} />
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="bg-slate-100 w-full flex-grow-1">
          <div className="p-8">
            {!(activeTab === "canvases" && workflows.length === 0 && !searchQuery) && (
              <PageHeader activeTab={activeTab} onNewClick={onNewClick} />
            )}

            {!(activeTab === "canvases" && workflows.length === 0 && !searchQuery) && (
              <>
                {showTabs && (
                  <Tabs
                    activeTab={activeTab}
                    setActiveTab={setActiveTab}
                    blueprints={filteredBlueprints}
                    workflows={filteredWorkflows}
                  />
                )}

                <div className="flex flex-col sm:flex-row gap-4 mb-6 justify-between">
                  <SearchBar activeTab={activeTab} searchQuery={searchQuery} setSearchQuery={setSearchQuery} />
                </div>
              </>
            )}

            {isLoading ? (
              <LoadingState activeTab={activeTab} />
            ) : error ? (
              <ErrorState error={error} />
            ) : (
              <Content
                activeTab={activeTab}
                filteredBlueprints={filteredBlueprints}
                filteredWorkflows={filteredWorkflows}
                organizationId={organizationId}
                searchQuery={searchQuery}
                onEditWorkflow={canvasModalState.onOpenEdit}
                onNewClick={onNewClick}
              />
            )}
          </div>
        </div>
      </main>

      <CreateCanvasModal {...canvasModalState} />
      {isCustomComponentsEnabled() && <CreateCustomComponentModal {...customComponentModalState} />}
    </div>
  );
};

//
// Tabs
//

interface TabsProps {
  activeTab: TabType;
  setActiveTab: (tab: TabType) => void;
  blueprints: BlueprintCardData[];
  workflows: WorkflowCardData[];
}

function Tabs({ activeTab, setActiveTab, blueprints, workflows }: TabsProps) {
  return (
    <div className="flex border-b border-border dark:border-gray-700 mb-6">
      <button
        onClick={() => setActiveTab("canvases")}
        className={`px-4 py-2 mb-[-1px] text-sm font-medium border-b transition-colors ${
          activeTab === "canvases"
            ? "border-gray-800 text-gray-800"
            : "border-transparent text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
        }`}
      >
        Canvases ({workflows.length})
      </button>

      {isCustomComponentsEnabled() && (
        <button
          onClick={() => setActiveTab("custom-components")}
          className={`px-4 py-2 mb-[-1px] text-sm font-medium border-b transition-colors ${
            activeTab === "custom-components"
              ? "border-gray-800 text-gray-800"
              : "border-transparent text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
          }`}
        >
          Bundles ({blueprints.length})
        </button>
      )}
    </div>
  );
}

interface SearchBarProps {
  activeTab: string;
  searchQuery: string;
  setSearchQuery: (query: string) => void;
}

function SearchBar({ activeTab, searchQuery, setSearchQuery }: SearchBarProps) {
  const searchPlaceholder = activeTab === "custom-components" ? "Search Bundles..." : "Search canvases...";

  return (
    <div className="flex items-center gap-2 w-full">
      <div className="flex-1 max-w-sm">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
          <Input
            placeholder={searchPlaceholder}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
      </div>
    </div>
  );
}

interface PageHeaderProps {
  activeTab: TabType;
  onNewClick: () => void;
}

function PageHeader({ activeTab, onNewClick }: PageHeaderProps) {
  const heading = activeTab === "custom-components" ? "Bundles" : "Canvases";
  const description =
    activeTab === "custom-components"
      ? "Bundles let you group multiple Components into a single reusable unit."
      : "Overview of all mapped automations across your organization.";
  const buttonText = activeTab === "custom-components" ? "New Bundle" : "New Canvas";

  return (
    <div className="flex items-center justify-between mb-6">
      <div>
        <Heading level={2} className="!text-2xl mb-1">
          {heading}
        </Heading>
        <Text className="text-gray-800 dark:text-gray-400">{description}</Text>
      </div>

      <Button onClick={onNewClick} size="sm">
        <Plus size={16} />
        {buttonText}
      </Button>
    </div>
  );
}

function LoadingState({ activeTab }: { activeTab: TabType }) {
  return (
    <div className="flex justify-center items-center h-40">
      <Text className="text-gray-500">Loading {activeTab}...</Text>
    </div>
  );
}

function ErrorState({ error }: { error: string }) {
  return (
    <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
      <Text>{error}</Text>
    </div>
  );
}

function Content({
  activeTab,
  filteredBlueprints,
  filteredWorkflows,
  organizationId,
  searchQuery,
  onEditWorkflow,
  onNewClick,
}: {
  activeTab: TabType;
  filteredBlueprints: BlueprintCardData[];
  filteredWorkflows: WorkflowCardData[];
  organizationId: string;
  searchQuery: string;
  onEditWorkflow: (workflow: WorkflowCardData) => void;
  onNewClick: () => void;
}) {
  if (activeTab === "canvases") {
    if (filteredWorkflows.length === 0) {
      return <CanvasesEmptyState searchQuery={searchQuery} onNewClick={onNewClick} />;
    }

    return (
      <WorkflowGridView
        filteredWorkflows={filteredWorkflows}
        organizationId={organizationId}
        onEditWorkflow={onEditWorkflow}
      />
    );
  } else if (activeTab === "custom-components") {
    if (filteredBlueprints.length === 0) {
      return <CustomComponentsEmptyState searchQuery={searchQuery} />;
    }

    return <BlueprintGridView filteredBlueprints={filteredBlueprints} organizationId={organizationId} />;
  }

  throw new Error("Invalid activeTab value");
}

function CustomComponentsEmptyState({ searchQuery }: { searchQuery: string }) {
  return (
    <div className="text-center py-12">
      <Box className="mx-auto text-gray-400 mb-4" size={48} />
      <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
        {searchQuery ? "No Bundles found" : "No Bundles yet"}
      </Heading>
      <Text className="text-gray-500 dark:text-gray-400 mb-6">
        {searchQuery ? "Try adjusting your search criteria." : "Get started by creating your first Bundle."}
      </Text>
    </div>
  );
}

function CanvasesEmptyState({ searchQuery, onNewClick }: { searchQuery: string; onNewClick: () => void }) {
  // Show different state when there's a search query vs when it's truly empty
  if (searchQuery) {
    return (
      <div className="text-center py-12">
        <GitBranch className="mx-auto text-gray-400 mb-4" size={48} />
        <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
          No canvases found
        </Heading>
        <Text className="text-gray-500 dark:text-gray-400 mb-6">
          Try adjusting your search criteria.
        </Text>
      </div>
    );
  }

  // Empty state when there are no canvases at all
  return (
    <div className="text-center py-12">
      <Palette className="mx-auto text-gray-800 dark:text-gray-300 mb-4" size={24} />
      <p className="text-sm text-gray-800 dark:text-gray-300 mb-6">
        Create your first Canvas
      </p>
      <Button onClick={onNewClick} size="sm">
        <Plus size={16} />
        New Canvas
      </Button>
    </div>
  );
}

interface WorkflowGridViewProps {
  filteredWorkflows: WorkflowCardData[];
  organizationId: string;
  onEditWorkflow: (workflow: WorkflowCardData) => void;
}

function WorkflowGridView({ filteredWorkflows, organizationId, onEditWorkflow }: WorkflowGridViewProps) {
  const navigate = useNavigate();

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
      {filteredWorkflows.map((workflow) => (
        <WorkflowCard
          key={workflow.id}
          workflow={workflow}
          organizationId={organizationId}
          navigate={navigate}
          onEdit={onEditWorkflow}
        />
      ))}
    </div>
  );
}

interface WorkflowCardProps {
  workflow: WorkflowCardData;
  organizationId: string;
  navigate: any;
  onEdit: (workflow: WorkflowCardData) => void;
}

function WorkflowCard({ workflow, organizationId, navigate, onEdit }: WorkflowCardProps) {
  const handleNavigate = () => navigate(`/${organizationId}/workflows/${workflow.id}`);
  const previewNodes = workflow.nodes || [];
  const previewEdges = workflow.edges || [];

  return (
    <div
      key={workflow.id}
      role="button"
      tabIndex={0}
      onClick={(event) => {
        if (event.defaultPrevented) return;
        handleNavigate();
      }}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          handleNavigate();
        }
      }}
      className="min-h-48 bg-white dark:bg-gray-950 rounded-md outline outline-slate-950/10 hover:shadow-md transition-shadow cursor-pointer"
    >
      <div className="flex flex-col h-full">
        <CanvasMiniMap nodes={previewNodes} edges={previewEdges} />

        <div className="p-4">
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col flex-1 min-w-0">
              <Heading
                level={3}
                className="!text-lg font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate"
              >
                <span className="truncate">{workflow.name}</span>
              </Heading>
            </div>
            <WorkflowActionsMenu workflow={workflow} organizationId={organizationId} onEdit={onEdit} />
          </div>

          {workflow.description ? (
            <div className="mb-4">
              <Text className="text-sm !leading-normal text-left text-gray-800 dark:text-gray-400 line-clamp-3">
                {workflow.description}
              </Text>
            </div>
          ) : null}

          <div className="flex justify-between items-center">
            <p className="text-xs text-gray-500 dark:text-gray-400 leading-none text-left mt-1">
              {workflow.createdBy?.name ? (
                <>
                  Created by {workflow.createdBy.name}, on {workflow.createdAt}
                </>
              ) : (
                <>Created on {workflow.createdAt}</>
              )}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

interface CanvasMiniMapProps {
  nodes?: ComponentsNode[];
  edges?: ComponentsEdge[];
}

function CanvasMiniMap({ nodes = [], edges = [] }: CanvasMiniMapProps) {
  const positionedNodes = nodes.filter(
    (node) => typeof node.position?.x === "number" && typeof node.position?.y === "number",
  ) as Array<ComponentsNode & { position: { x: number; y: number } }>;

  if (!positionedNodes.length) {
    return (
      <div className="p-4 border-b border-gray-200">
        <div className="h-28 w-full rounded-sm border border-dashed border-gray-300 dark:border-gray-700 bg-transparent flex items-center justify-center text-[13px] text-gray-500 dark:text-gray-400">
          The canvas is empty
        </div>
      </div>
    );
  }

  const xs = positionedNodes.map((node) => node.position.x);
  const ys = positionedNodes.map((node) => node.position.y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const padding = 80;
  const width = Math.max(maxX - minX, 200) + padding * 2;
  const height = Math.max(maxY - minY, 200) + padding * 2;
  const viewBox = `${minX - padding} ${minY - padding} ${width} ${height}`;
  const nodeWidth = Math.min(Math.max(width * 0.08, 30), 80);
  const nodeHeight = nodeWidth * 0.45;

  const nodePositions = new Map<string, { x: number; y: number }>();
  positionedNodes.forEach((node) => {
    const id = node.id || node.name;
    if (!id) return;
    nodePositions.set(id, { x: node.position.x, y: node.position.y });
  });

  const drawableEdges =
    edges?.filter(
      (edge) => edge.sourceId && edge.targetId && nodePositions.has(edge.sourceId) && nodePositions.has(edge.targetId),
    ) || [];

  return (
    <div className="p-4 w-full border-b border-gray-200 overflow-hidden">
      <svg
        viewBox={viewBox}
        preserveAspectRatio="xMidYMid meet"
        className="w-full h-28 text-gray-500 dark:text-gray-400"
      >
        {drawableEdges.map((edge) => {
          const source = nodePositions.get(edge.sourceId!);
          const target = nodePositions.get(edge.targetId!);
          if (!source || !target) return null;
          return (
            <line
              key={`${edge.sourceId}-${edge.targetId}`}
              x1={source.x}
              y1={source.y}
              x2={target.x}
              y2={target.y}
              stroke="currentColor"
              strokeWidth={6}
              strokeLinecap="round"
              opacity={0.25}
            />
          );
        })}
        {positionedNodes.map((node) => {
          const id = node.id || node.name || `${node.position.x}-${node.position.y}`;
          return (
            <rect
              key={id}
              x={node.position.x - nodeWidth / 2}
              y={node.position.y - nodeHeight / 2}
              width={nodeWidth}
              height={nodeHeight}
              rx={8}
              ry={8}
              fill="#1f2937"
              opacity={1}
            />
          );
        })}
      </svg>
    </div>
  );
}

interface WorkflowActionsMenuProps {
  workflow: WorkflowCardData;
  organizationId: string;
  onEdit: (workflow: WorkflowCardData) => void;
}

function WorkflowActionsMenu({ workflow, organizationId, onEdit }: WorkflowActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const deleteWorkflowMutation = useDeleteWorkflow(organizationId);
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleDelete = async () => {
    // If we're currently viewing this workflow, navigate immediately and remove from cache to prevent 404
    const currentPath = location.pathname;
    const workflowPath = `/${organizationId}/workflows/${workflow.id}`;
    const isViewingWorkflow = currentPath === workflowPath || currentPath.startsWith(`${workflowPath}/`);
    
    if (isViewingWorkflow) {
      // Remove from cache FIRST to prevent any queries from running
      queryClient.removeQueries({ queryKey: workflowKeys.detail(organizationId, workflow.id) });
      // Navigate immediately with replace to avoid back button issues and prevent 404 flash
      navigate(`/${organizationId}`, { replace: true });
      // Then delete (fire and forget)
      deleteWorkflowMutation.mutate(workflow.id, {
        onSuccess: () => {
          showSuccessToast("Canvas deleted successfully");
          closeDialog();
        },
        onError: (error) => {
          console.error("Failed to delete canvas:", error);
          showErrorToast("Failed to delete canvas");
        },
      });
      return;
    }
    
    try {
      await deleteWorkflowMutation.mutateAsync(workflow.id);
      showSuccessToast("Canvas deleted successfully");
      closeDialog();
    } catch (error) {
      console.error("Failed to delete canvas:", error);
      showErrorToast("Failed to delete canvas");
    }
  };

  return (
    <>
      <div className="flex-shrink-0" onClick={(event: MouseEvent<HTMLDivElement>) => event.stopPropagation()}>
        <DropdownMenu>
          <DropdownMenuTrigger
            asChild
            onClick={(event: MouseEvent<HTMLButtonElement>) => event.stopPropagation()}
          >
            <button
              className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
              aria-label="Canvas actions"
              disabled={deleteWorkflowMutation.isPending}
            >
              <MoreVertical size={16} />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={(event: MouseEvent<HTMLElement>) => {
                event.stopPropagation();
                onEdit(workflow);
              }}
            >
              <Pencil size={16} />
              Edit
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={openDialog}
              className="text-red-600 dark:text-red-400 focus:text-red-600 dark:focus:text-red-400"
            >
              <Trash2 size={16} />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Dialog open={isDialogOpen} onClose={closeDialog} size="lg" className="text-left">
        <DialogTitle className="text-gray-800 dark:text-red-100">Delete "{workflow.name}"?</DialogTitle>
        <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
          This cannot be undone. Are you sure you want to continue?
        </DialogDescription>
        <DialogActions>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteWorkflowMutation.isPending}
            className="flex items-center gap-2"
          >
            <Trash2 size={16} />
            {deleteWorkflowMutation.isPending ? "Deleting..." : "Delete"}
          </Button>
          <Button variant="outline" onClick={closeDialog}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

interface BlueprintActionsMenuProps {
  blueprint: BlueprintCardData;
  organizationId: string;
}

function BlueprintActionsMenu({ blueprint, organizationId }: BlueprintActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const deleteBlueprintMutation = useDeleteBlueprint(organizationId);
  const navigate = useNavigate();

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleDelete = async () => {
    try {
      await deleteBlueprintMutation.mutateAsync(blueprint.id);
      showSuccessToast("Component deleted successfully");
      closeDialog();
    } catch (error) {
      console.error("Failed to delete Bundle:", error);
      showErrorToast("Failed to delete Bundle");
    }
  };

  return (
    <>
      <div className="flex-shrink-0" onClick={(event: MouseEvent<HTMLDivElement>) => event.stopPropagation()}>
        <DropdownMenu>
          <DropdownMenuTrigger
            asChild
            onClick={(event: MouseEvent<HTMLButtonElement>) => event.stopPropagation()}
          >
            <button
              className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
              aria-label="Component actions"
              disabled={deleteBlueprintMutation.isPending}
            >
              <MoreVertical size={16} />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onClick={(event: MouseEvent<HTMLElement>) => {
                event.stopPropagation();
                navigate(`/${organizationId}/custom-components/${blueprint.id}`);
              }}
            >
              <Pencil size={16} />
              Edit
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={openDialog}
              className="text-red-600 dark:text-red-400 focus:text-red-600 dark:focus:text-red-400"
            >
              <Trash2 size={16} />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Dialog open={isDialogOpen} onClose={closeDialog} size="lg" className="text-left">
        <DialogTitle className="text-gray-800 dark:text-red-100">Delete "{blueprint.name}"?</DialogTitle>
        <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
          This cannot be undone. Are you sure you want to continue?
        </DialogDescription>
        <DialogActions>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteBlueprintMutation.isPending}
            className="flex items-center gap-2"
          >
            <Trash2 size={16} />
            {deleteBlueprintMutation.isPending ? "Deleting..." : "Delete"}
          </Button>
          <Button variant="outline" onClick={closeDialog}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

interface BlueprintGridViewProps {
  filteredBlueprints: BlueprintCardData[];
  organizationId: string;
}

function BlueprintGridView({ filteredBlueprints, organizationId }: BlueprintGridViewProps) {
  const navigate = useNavigate();

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
      {filteredBlueprints.map((blueprint) => {
        const IconComponent = resolveIcon("component");
        const handleNavigate = () => navigate(`/${organizationId}/custom-components/${blueprint.id}`);
        return (
          <div
            key={blueprint.id}
            role="button"
            tabIndex={0}
            onClick={(event) => {
              if (event.defaultPrevented) return;
              handleNavigate();
            }}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                handleNavigate();
              }
            }}
            className="min-h-48 bg-white dark:bg-gray-950 rounded-md outline outline-slate-950/10 dark:border-gray-800 hover:shadow-md transition-shadow cursor-pointer"
          >
            <div className="p-6 flex flex-col justify-between h-full">
              <div>
                <div className="flex items-start justify-between gap-3 mb-4">
                  <div className="flex items-center space-x-3 flex-1 min-w-0">
                    <IconComponent size={16} className="text-gray-800 dark:text-gray-400" />
                    <div className="flex flex-col flex-1 min-w-0">
                      <Heading
                        level={3}
                        className="!text-lg font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate"
                      >
                        {blueprint.name}
                      </Heading>
                    </div>
                  </div>
                  <BlueprintActionsMenu blueprint={blueprint} organizationId={organizationId} />
                </div>

                {blueprint.description ? (
                  <div className="mb-4">
                    <Text className="text-sm text-left text-gray-800 dark:text-gray-400 line-clamp-3">
                      {blueprint.description}
                    </Text>
                  </div>
                ) : null}
              </div>

              <div className="flex justify-between items-center">
                <p className="text-xs text-gray-500 dark:text-gray-400 leading-none text-left">
                  {blueprint.createdBy?.name ? (
                    <>
                      By {blueprint.createdBy.name}, created on {blueprint.createdAt}
                    </>
                  ) : (
                    <>Created on {blueprint.createdAt}</>
                  )}
                </p>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default HomePage;
