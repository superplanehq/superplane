import { Box, GitBranch, LayoutGrid, List, MoreVertical, Plus, Search, Trash2 } from "lucide-react";
import { useState, type MouseEvent } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../../components/Button/button";
import { CreateCanvasModal } from "../../components/CreateCanvasModal";
import { CreateCustomComponentModal } from "../../components/CreateCustomComponentModal";
import { Heading } from "../../components/Heading/heading";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/AccountContext";
import { useBlueprints } from "../../hooks/useBlueprintData";
import { useDeleteWorkflow, useWorkflows } from "../../hooks/useWorkflowData";
import { cn, resolveIcon } from "../../lib/utils";
import { getColorClass } from "../../utils/colors";
import { showErrorToast, showSuccessToast } from "../../utils/toast";
import { Dropdown, DropdownButton, DropdownItem, DropdownMenu } from "../../components/Dropdown/dropdown";
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from "../../components/Dialog/dialog";

import { useCreateCanvasModalState } from "./useCreateCanvasModalState";
import { useCreateCustomComponentModalState } from "./useCreateCustomComponentModalState";

type TabType = "canvases" | "custom-components";
type ViewMode = "grid" | "list";

interface BlueprintCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "blueprint";
  icon?: string;
  color?: string;
  createdBy?: { id?: string; name?: string };
}

interface WorkflowCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "workflow";
  createdBy?: { id?: string; name?: string };
}

const HomePage = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const [viewMode, setViewMode] = useState<ViewMode>("grid");
  const [activeTab, setActiveTab] = useState<TabType>("canvases");

  const canvasModalState = useCreateCanvasModalState();
  const customComponentModalState = useCreateCustomComponentModalState();

  const { organizationId } = useParams<{ organizationId: string }>();
  const { account } = useAccount();

  const {
    data: blueprintsData = [],
    isLoading: blueprintsLoading,
    error: blueprintApiError,
  } = useBlueprints(organizationId || "");

  const {
    data: workflowsData = [],
    isLoading: workflowsLoading,
    error: workflowApiError,
  } = useWorkflows(organizationId || "");

  const blueprintError = blueprintApiError ? "Failed to fetch components. Please try again later." : null;
  const workflowError = workflowApiError ? "Failed to fetch workflows. Please try again later." : null;

  const blueprints: BlueprintCardData[] = (blueprintsData || []).map((blueprint: any) => ({
    id: blueprint.id!,
    name: blueprint.name!,
    description: blueprint.description,
    createdAt: blueprint.createdAt ? new Date(blueprint.createdAt).toLocaleDateString() : "Unknown",
    type: "blueprint" as const,
    icon: blueprint.icon,
    color: blueprint.color,
    createdBy: blueprint.createdBy,
  }));

  const workflows: WorkflowCardData[] = (workflowsData || []).map((workflow: any) => ({
    id: workflow.id!,
    name: workflow.name!,
    description: workflow.description,
    createdAt: workflow.createdAt ? new Date(workflow.createdAt).toLocaleDateString() : "Unknown",
    type: "workflow" as const,
    createdBy: workflow.createdBy,
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
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
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
    if (activeTab === "custom-components") {
      customComponentModalState.onOpen();
    } else {
      canvasModalState.onOpen();
    }
  };

  return (
    <div className="min-h-screen flex flex-col bg-zinc-50 dark:bg-zinc-900 pt-10">
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="bg-zinc-50 dark:bg-zinc-900 w-full flex-grow-1 p-6">
          <div className="p-4">
            <PageHeader activeTab={activeTab} onNewClick={onNewClick} />

            <Tabs
              activeTab={activeTab}
              setActiveTab={setActiveTab}
              blueprints={filteredBlueprints}
              workflows={filteredWorkflows}
            />

            <div className="flex flex-col sm:flex-row gap-4 mb-6 justify-between">
              <SearchBar activeTab={activeTab} searchQuery={searchQuery} setSearchQuery={setSearchQuery} />
              <ViewModeToggle viewMode={viewMode} setViewMode={setViewMode} />
            </div>

            {isLoading ? (
              <LoadingState activeTab={activeTab} />
            ) : error ? (
              <ErrorState error={error} />
            ) : (
              <Content
                activeTab={activeTab}
                viewMode={viewMode}
                filteredBlueprints={filteredBlueprints}
                filteredWorkflows={filteredWorkflows}
                organizationId={organizationId}
                searchQuery={searchQuery}
              />
            )}
          </div>
        </div>
      </main>

      <CreateCanvasModal {...canvasModalState} />
      <CreateCustomComponentModal {...customComponentModalState} />
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
    <div className="flex border-b border-zinc-200 dark:border-zinc-700 mb-6">
      <button
        onClick={() => setActiveTab("canvases")}
        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
          activeTab === "canvases"
            ? "border-blue-600 text-blue-600"
            : "border-transparent text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
        }`}
      >
        Canvases ({workflows.length})
      </button>

      <button
        onClick={() => setActiveTab("custom-components")}
        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
          activeTab === "custom-components"
            ? "border-blue-600 text-blue-600"
            : "border-transparent text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
        }`}
      >
        Components ({blueprints.length})
      </button>
    </div>
  );
}

interface ViewModeToggleProps {
  viewMode: string;
  setViewMode: any;
}

function ViewModeToggle({ viewMode, setViewMode }: ViewModeToggleProps) {
  return (
    <div className="flex items-center">
      <Button
        {...(viewMode === "grid" ? { color: "light" as const } : { plain: true })}
        onClick={() => setViewMode("grid")}
        title="Grid view"
      >
        <LayoutGrid size={18} />
      </Button>
      <Button
        {...(viewMode === "list" ? { color: "light" as const } : { plain: true })}
        onClick={() => setViewMode("list")}
        title="List view"
      >
        <List size={18} />
      </Button>
    </div>
  );
}

interface SearchBarProps {
  activeTab: string;
  searchQuery: string;
  setSearchQuery: (query: string) => void;
}

function SearchBar({ activeTab, searchQuery, setSearchQuery }: SearchBarProps) {
  const inputStyle = cn(
    "h-9 w-full pl-10 pr-4 py-2 border border-zinc-200 dark:border-zinc-700",
    "rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder-zinc-500",
    "focus:ring-2 focus:ring-blue-500 focus:border-transparent",
  );

  const searchPlaceholder = activeTab === "custom-components" ? "Search components..." : "Search canvases...";

  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 w-100">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-400" size={18} />
          <input
            type="text"
            placeholder={searchPlaceholder}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className={inputStyle}
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
  const heading = activeTab === "custom-components" ? "Components" : "Canvases";
  const description = activeTab === "custom-components"
    ? "Overview of all components created and maintained by your team."
    : "Overview of all mapped automations across your organization.";
  const buttonText = activeTab === "custom-components" ? "New Component" : "New Canvas";

  return (
    <div className="flex items-center justify-between mb-6">
      <div>
        <Heading level={2} className="!text-2xl mb-2">
          {heading}
        </Heading>
        <Text className="text-zinc-600 dark:text-zinc-400">
          {description}
        </Text>
      </div>

      <Button color="blue" className="flex items-center bg-blue-700 text-white hover:bg-blue-600" onClick={onNewClick}>
        <Plus size={20} />
        {buttonText}
      </Button>
    </div>
  );
}

function LoadingState({ activeTab }: { activeTab: TabType }) {
  return (
    <div className="flex justify-center items-center h-40">
      <Text className="text-zinc-500">Loading {activeTab}...</Text>
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
  viewMode,
  filteredBlueprints,
  filteredWorkflows,
  organizationId,
  searchQuery,
}: {
  activeTab: TabType;
  viewMode: ViewMode;
  filteredBlueprints: BlueprintCardData[];
  filteredWorkflows: WorkflowCardData[];
  organizationId: string;
  searchQuery: string;
}) {
  if (activeTab === "canvases") {
    if (filteredWorkflows.length === 0) {
      return <CanvasesEmptyState searchQuery={searchQuery} />;
    }

    if (viewMode === "grid") {
      return <WorkflowGridView filteredWorkflows={filteredWorkflows} organizationId={organizationId} />;
    } else {
      return <WorkflowListView filteredWorkflows={filteredWorkflows} organizationId={organizationId} />;
    }
  } else if (activeTab === "custom-components") {
    if (filteredBlueprints.length === 0) {
      return <CustomComponentsEmptyState searchQuery={searchQuery} />;
    }

    if (viewMode === "grid") {
      return <BlueprintGridView filteredBlueprints={filteredBlueprints} organizationId={organizationId} />;
    } else {
      return <BlueprintListView filteredBlueprints={filteredBlueprints} organizationId={organizationId} />;
    }
  }

  throw new Error("Invalid activeTab value");
}

function CustomComponentsEmptyState({ searchQuery }: { searchQuery: string }) {
  return (
    <div className="text-center py-12">
      <Box className="mx-auto text-zinc-400 mb-4" size={48} />
      <Heading level={3} className="text-lg text-zinc-900 dark:text-white mb-2">
        {searchQuery ? "No components found" : "No components yet"}
      </Heading>
      <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
        {searchQuery ? "Try adjusting your search criteria." : "Get started by creating your first component."}
      </Text>
    </div>
  );
}

function CanvasesEmptyState({ searchQuery }: { searchQuery: string }) {
  return (
    <div className="text-center py-12">
      <GitBranch className="mx-auto text-zinc-400 mb-4" size={48} />
      <Heading level={3} className="text-lg text-zinc-900 dark:text-white mb-2">
        {searchQuery ? "No canvases found" : "No canvases yet"}
      </Heading>
      <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
        {searchQuery ? "Try adjusting your search criteria." : "Get started by creating your first canvas."}
      </Text>
    </div>
  );
}

interface WorkflowGridViewProps {
  filteredWorkflows: WorkflowCardData[];
  organizationId: string;
}

function WorkflowGridView({ filteredWorkflows, organizationId }: WorkflowGridViewProps) {
  const navigate = useNavigate();

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6">
      {filteredWorkflows.map((workflow) => (
        <WorkflowCard key={workflow.id} workflow={workflow} organizationId={organizationId} navigate={navigate} />
      ))}
    </div>
  );
}

interface WorkflowListViewProps {
  filteredWorkflows: WorkflowCardData[];
  organizationId: string;
}

function WorkflowListView({ filteredWorkflows, organizationId }: WorkflowListViewProps) {
  const navigate = useNavigate();

  return (
    <div className="space-y-2">
      {filteredWorkflows.map((workflow) => (
        <WorkflowListItem key={workflow.id} workflow={workflow} organizationId={organizationId} navigate={navigate} />
      ))}
    </div>
  );
}

interface WorkflowCardProps {
  workflow: WorkflowCardData;
  organizationId: string;
  navigate: any;
}

function WorkflowCard({ workflow, organizationId, navigate }: WorkflowCardProps) {
  return (
    <div
      key={workflow.id}
      className="max-h-45 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-md transition-shadow"
    >
      <div className="p-6 flex flex-col justify-between h-full">
        <div>
          <div className="flex items-start justify-between gap-3 mb-4">
            <div className="flex flex-col flex-1 min-w-0">
              <button
                onClick={() => navigate(`/${organizationId}/workflows/${workflow.id}`)}
                className="block text-left w-full"
              >
                <Heading
                  level={3}
                  className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate"
                >
                  {workflow.name}
                </Heading>
              </button>
            </div>
            <WorkflowActionsMenu workflow={workflow} organizationId={organizationId} />
          </div>

          <div className="mb-4">
            <Text className="text-sm text-left text-zinc-600 dark:text-zinc-400 line-clamp-2 mt-2">
              {workflow.description || "No description"}
            </Text>
          </div>
        </div>

        <div className="flex justify-between items-center">
          <div className="text-zinc-500 text-left">
            {workflow.createdBy?.name && (
              <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none mb-1">Created by <strong>{workflow.createdBy.name}</strong></p>
            )}
            <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none">Created at {workflow.createdAt}</p>
          </div>
        </div>
      </div>
    </div>
  );
}

function WorkflowListItem({ workflow, organizationId, navigate }: WorkflowCardProps) {
  return (
    <div
      key={workflow.id}
      className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-sm transition-shadow p-4"
    >
      <div className="flex items-start justify-between gap-3">
        <button
          onClick={() => navigate(`/${organizationId}/workflows/${workflow.id}`)}
          className="block text-left w-full"
        >
          <Heading
            level={3}
            className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-1"
          >
            {workflow.name}
          </Heading>
          <Text className="text-sm text-zinc-600 dark:text-zinc-400">{workflow.description || "No description"}</Text>
          <Text className="text-xs text-zinc-500 mt-2">
            {workflow.createdBy?.name ? (
              <>
                Created by <strong>{workflow.createdBy.name}</strong> · {workflow.createdAt}
              </>
            ) : (
              <>Created at {workflow.createdAt}</>
            )}
          </Text>
        </button>

        <WorkflowActionsMenu workflow={workflow} organizationId={organizationId} />
      </div>
    </div>
  );
}

interface WorkflowActionsMenuProps {
  workflow: WorkflowCardData;
  organizationId: string;
}

function WorkflowActionsMenu({ workflow, organizationId }: WorkflowActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const deleteWorkflowMutation = useDeleteWorkflow(organizationId);

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleDelete = async () => {
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
        <Dropdown>
          <DropdownButton
            plain
            className="p-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-500 dark:text-zinc-400"
            aria-label="Canvas actions"
            onClick={(event: MouseEvent<HTMLButtonElement>) => event.stopPropagation()}
            disabled={deleteWorkflowMutation.isPending}
          >
            <MoreVertical size={16} />
          </DropdownButton>
          <DropdownMenu>
            <DropdownItem
              onClick={openDialog}
              className="text-red-600 dark:text-red-400"
            >
              <span className="flex items-center gap-2">
                <Trash2 size={16} />
                Delete
              </span>
            </DropdownItem>
          </DropdownMenu>
        </Dropdown>
      </div>

      <Dialog open={isDialogOpen} onClose={closeDialog} size="lg" className="text-left">
        <DialogTitle className="text-red-900 dark:text-red-100">Delete canvas</DialogTitle>
        <DialogDescription className="text-sm text-zinc-600 dark:text-zinc-400">
          This action cannot be undone. Are you sure you want to delete this canvas?
        </DialogDescription>
        <DialogBody>
          <Text className="text-sm text-zinc-600 dark:text-zinc-400">
            Deleting <span className="font-medium text-zinc-900 dark:text-zinc-100">{workflow.name}</span> will remove its automations and history.
          </Text>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={closeDialog}>Cancel</Button>
          <Button
            color="red"
            onClick={handleDelete}
            disabled={deleteWorkflowMutation.isPending}
            className="flex items-center gap-2"
          >
            <Trash2 size={16} />
            {deleteWorkflowMutation.isPending ? "Deleting..." : "Delete"}
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
    <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6">
      {filteredBlueprints.map((blueprint) => {
        const IconComponent = resolveIcon(blueprint.icon);
        return (
          <div
            key={blueprint.id}
            className="max-h-45 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-md transition-shadow"
          >
            <div className="p-6 flex flex-col justify-between h-full">
              <div>
                <div className="flex items-center mb-4">
                  <div className="flex items-center justify-between space-x-3 flex-1">
                    <IconComponent size={24} className={getColorClass(blueprint.color)} />
                    <div className="flex flex-col flex-1 min-w-0">
                      <button
                        onClick={() => navigate(`/${organizationId}/custom-components/${blueprint.id}`)}
                        className="block text-left w-full"
                      >
                        <Heading
                          level={3}
                          className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate"
                        >
                          {blueprint.name}
                        </Heading>
                      </button>
                    </div>
                  </div>
                </div>

                <div className="mb-4">
                  <Text className="text-sm text-left text-zinc-600 dark:text-zinc-400 line-clamp-2 mt-2">
                    {blueprint.description || "No description"}
                  </Text>
                </div>
              </div>

              <div className="flex justify-between items-center">
                <div className="text-zinc-500 text-left">
                  {blueprint.createdBy?.name && (
                    <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none mb-1">
                      Created by <strong>{blueprint.createdBy.name}</strong>
                    </p>
                  )}
                  <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none">
                    Created at {blueprint.createdAt}
                  </p>
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function BlueprintListView({ filteredBlueprints, organizationId }: BlueprintGridViewProps) {
  const navigate = useNavigate();

  return (
    <div className="space-y-2">
      {filteredBlueprints.map((blueprint) => {
        const IconComponent = resolveIcon(blueprint.icon);
        return (
          <div
            key={blueprint.id}
            className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-sm transition-shadow p-4"
          >
            <button
              onClick={() => navigate(`/${organizationId}/custom-components/${blueprint.id}`)}
              className="block text-left w-full"
            >
              <div className="flex items-center gap-3">
                <IconComponent size={24} className={getColorClass(blueprint.color)} />
                <div className="flex-1">
                  <Heading
                    level={3}
                    className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-1"
                  >
                    {blueprint.name}
                  </Heading>
                  <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                    {blueprint.description || "No description"}
                  </Text>
                  <Text className="text-xs text-zinc-500 mt-2">
                    {blueprint.createdBy?.name ? (
                      <>
                        Created by <strong>{blueprint.createdBy.name}</strong> · {blueprint.createdAt}
                      </>
                    ) : (
                      <>Created at {blueprint.createdAt}</>
                    )}
                  </Text>
                </div>
              </div>
            </button>
          </div>
        );
      })}
    </div>
  );
}

export default HomePage;
