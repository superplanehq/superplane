import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Grid3x3, MoreVertical, Pencil, Plus, Palette, Rainbow, Rows3, Search, Trash2 } from "lucide-react";
import { useState, type MouseEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { CreateCanvasModal } from "../../components/CreateCanvasModal";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "../../components/Dialog/dialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "../../ui/dropdownMenu";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/AccountContext";
import { usePermissions } from "@/contexts/PermissionsContext";
import { PermissionTooltip } from "@/components/PermissionGate";
import { useDeleteCanvas, useCanvases, canvasKeys } from "../../hooks/useCanvasData";
import { cn } from "../../lib/utils";
import { showErrorToast, showSuccessToast } from "../../lib/toast";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useCreateCanvasModalState } from "./useCreateCanvasModalState";
import { OnboardingWelcome } from "./OnboardingWelcome";
import type { ComponentsEdge, ComponentsNode } from "@/api-client";

type CanvasViewMode = "grid" | "list";

interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "canvases";
  createdBy?: { id?: string; name?: string };
  nodes?: ComponentsNode[];
  edges?: ComponentsEdge[];
}

const HomePage = () => {
  usePageTitle(["Home"]);

  const [searchQuery, setSearchQuery] = useState("");
  const [canvasViewMode, setCanvasViewMode] = useState<CanvasViewMode>("grid");
  const canvasModalState = useCreateCanvasModalState();

  const { organizationId } = useParams<{ organizationId: string }>();
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const {
    data: canvasesData = [],
    isLoading: canvasesLoading,
    error: canvasesApiError,
  } = useCanvases(organizationId || "");

  const canvasError = canvasesApiError ? "Failed to fetch canvases. Please try again later." : null;
  const canCreateCanvases = canAct("canvases", "create");
  const canUpdateCanvases = canAct("canvases", "update");
  const canDeleteCanvases = canAct("canvases", "delete");

  const formatDate = (value?: string) => {
    if (!value) return "Unknown";
    return new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  };

  const canvases: CanvasCardData[] = (canvasesData || []).map((canvas: any) => ({
    id: canvas.metadata?.id!,
    name: canvas.metadata?.name!,
    description: canvas.metadata?.description,
    createdAt: formatDate(canvas.metadata?.createdAt),
    type: "canvases" as const,
    createdBy: canvas.metadata?.createdBy,
    nodes: canvas.spec?.nodes || [],
    edges: canvas.spec?.edges || [],
  }));

  const filteredCanvases = canvases.filter((canvas) => {
    const matchesSearch =
      canvas.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      canvas.description?.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesSearch;
  });

  const isLoading = canvasesLoading;

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
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

  const error = canvasError;

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-slate-900">
      <header className="bg-white border-b border-slate-950/15 px-4 h-12 flex items-center">
        <OrganizationMenuButton organizationId={organizationId} />
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="bg-slate-100 w-full flex-grow-1">
          <div className="mx-auto w-full max-w-6xl p-8">
            <div className="flex items-center justify-between mb-6">
              <div>
                <Heading level={2} className="!text-2xl mb-1">
                  Canvases
                </Heading>
                <Text className="text-gray-800 dark:text-gray-400">
                  Overview of all mapped automations across your organization.
                </Text>
              </div>
            </div>

            {!(canvases.length === 0 && !searchQuery) && (
              <div className="mb-6">
                <SearchBar
                  searchQuery={searchQuery}
                  setSearchQuery={setSearchQuery}
                  canvasViewMode={canvasViewMode}
                  onCanvasViewModeChange={setCanvasViewMode}
                />
              </div>
            )}

            {isLoading ? (
              <div className="flex justify-center items-center h-40">
                <Text className="text-gray-500">Loading canvases...</Text>
              </div>
            ) : error ? (
              <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded">
                <Text>{error}</Text>
              </div>
            ) : (
              <Content
                filteredCanvases={filteredCanvases}
                organizationId={organizationId}
                searchQuery={searchQuery}
                canvasViewMode={canvasViewMode}
                onEditCanvas={canvasModalState.onOpenEdit}
                canCreateCanvases={canCreateCanvases}
                canUpdateCanvases={canUpdateCanvases}
                canDeleteCanvases={canDeleteCanvases}
                permissionsLoading={permissionsLoading}
              />
            )}
          </div>
        </div>
      </main>

      <CreateCanvasModal {...canvasModalState} />
    </div>
  );
};

interface SearchBarProps {
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  canvasViewMode: CanvasViewMode;
  onCanvasViewModeChange: (mode: CanvasViewMode) => void;
}

function SearchBar({ searchQuery, setSearchQuery, canvasViewMode, onCanvasViewModeChange }: SearchBarProps) {
  return (
    <div className="flex w-full flex-wrap items-center justify-between gap-4">
      <div className="min-w-0 w-full shrink-0 md:w-[calc((100%-1.5rem)/2)] lg:w-[calc((100%-3rem)/3)]">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
          <Input
            placeholder="Filter canvases…"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
      </div>
      <div className="ml-auto flex shrink-0 items-center">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "h-7 w-7 bg-transparent p-0 hover:bg-transparent dark:hover:bg-transparent",
            canvasViewMode === "grid" ? "opacity-100" : "opacity-50 hover:opacity-100",
          )}
          aria-label="Grid view"
          aria-pressed={canvasViewMode === "grid"}
          onClick={() => onCanvasViewModeChange("grid")}
        >
          <Grid3x3 className="h-3.5 w-3.5" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "h-7 w-7 bg-transparent p-0 hover:bg-transparent dark:hover:bg-transparent",
            canvasViewMode === "list" ? "opacity-100" : "opacity-50 hover:opacity-100",
          )}
          aria-label="List view"
          aria-pressed={canvasViewMode === "list"}
          onClick={() => onCanvasViewModeChange("list")}
        >
          <Rows3 className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}

function Content({
  filteredCanvases,
  organizationId,
  searchQuery,
  canvasViewMode,
  onEditCanvas,
  canCreateCanvases,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: {
  filteredCanvases: CanvasCardData[];
  organizationId: string;
  searchQuery: string;
  canvasViewMode: CanvasViewMode;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canCreateCanvases: boolean;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}) {
  if (filteredCanvases.length === 0) {
    if (searchQuery) {
      return <CanvasesSearchEmptyState />;
    }
    return (
      <OnboardingWelcome
        organizationId={organizationId}
        canCreateCanvases={canCreateCanvases}
        permissionsLoading={permissionsLoading}
      />
    );
  }

  return (
    <CanvasGridView
      filteredCanvases={filteredCanvases}
      organizationId={organizationId}
      view={canvasViewMode}
      searchQuery={searchQuery}
      onEditCanvas={onEditCanvas}
      canCreateCanvases={canCreateCanvases}
      canUpdateCanvases={canUpdateCanvases}
      canDeleteCanvases={canDeleteCanvases}
      permissionsLoading={permissionsLoading}
    />
  );
}

function CanvasesSearchEmptyState() {
  return (
    <div className="text-center py-12">
      <Palette className="mx-auto text-gray-400 mb-4" size={48} aria-hidden />
      <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
        No canvases found
      </Heading>
      <Text className="text-gray-500 dark:text-gray-400 mb-6">
        Nothing matches that filter, try another word or clear it
      </Text>
    </div>
  );
}

interface CanvasGridViewProps {
  filteredCanvases: CanvasCardData[];
  organizationId: string;
  view: CanvasViewMode;
  searchQuery: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canCreateCanvases: boolean;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function NewCanvasCard({
  organizationId,
  canCreateCanvases,
  permissionsLoading,
}: {
  organizationId: string;
  canCreateCanvases: boolean;
  permissionsLoading: boolean;
}) {
  const allowed = canCreateCanvases || permissionsLoading;

  return (
    <PermissionTooltip allowed={allowed} message="You don't have permission to create canvases." className="min-w-0">
      <Link
        to={`/${organizationId}/canvases/new`}
        aria-label="Create new canvas"
        className={cn(
          "relative flex min-h-48 flex-col items-center justify-center rounded-md border border-dashed border-green-500 bg-green-50 text-center transition-colors dark:border-green-500 dark:bg-green-950/30",
          "hover:bg-green-100 dark:hover:bg-green-950/50",
          allowed && "cursor-pointer",
        )}
      >
        <div className="flex flex-col items-center justify-center gap-3 px-4">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-green-500 text-white">
            <Plus className="h-4 w-4" strokeWidth={2} aria-hidden />
          </span>
          <Heading
            level={3}
            className="!text-base font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate"
          >
            <span className="truncate">New Canvas</span>
          </Heading>
        </div>
      </Link>
    </PermissionTooltip>
  );
}

function NewCanvasListRow({
  organizationId,
  canCreateCanvases,
  permissionsLoading,
}: {
  organizationId: string;
  canCreateCanvases: boolean;
  permissionsLoading: boolean;
}) {
  const allowed = canCreateCanvases || permissionsLoading;

  return (
    <PermissionTooltip allowed={allowed} message="You don't have permission to create canvases." className="min-w-0">
      <Link
        to={`/${organizationId}/canvases/new`}
        aria-label="Create new canvas"
        className={cn(
          "relative flex flex-row items-center gap-4 rounded-md border border-dashed border-green-500 bg-green-50 px-4 py-3 transition-colors dark:border-green-500 dark:bg-green-950/30",
          "hover:bg-green-100 dark:hover:bg-green-950/50",
          allowed && "cursor-pointer",
        )}
      >
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-green-500 text-white">
          <Plus className="h-4 w-4" strokeWidth={2} aria-hidden />
        </span>
        <Heading
          level={3}
          className="!text-base font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 truncate"
        >
          <span className="truncate">New Canvas</span>
        </Heading>
      </Link>
    </PermissionTooltip>
  );
}

function CanvasGridView({
  filteredCanvases,
  organizationId,
  view,
  searchQuery,
  onEditCanvas,
  canCreateCanvases,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasGridViewProps) {
  const showNewCanvasEntry = searchQuery.trim() === "";

  if (view === "list") {
    return (
      <div className="flex flex-col gap-3">
        {showNewCanvasEntry ? (
          <NewCanvasListRow
            organizationId={organizationId}
            canCreateCanvases={canCreateCanvases}
            permissionsLoading={permissionsLoading}
          />
        ) : null}
        {filteredCanvases.map((canvas) => (
          <CanvasListRow
            key={canvas.id}
            canvas={canvas}
            organizationId={organizationId}
            onEdit={onEditCanvas}
            canUpdateCanvases={canUpdateCanvases}
            canDeleteCanvases={canDeleteCanvases}
            permissionsLoading={permissionsLoading}
          />
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {showNewCanvasEntry ? (
        <NewCanvasCard
          organizationId={organizationId}
          canCreateCanvases={canCreateCanvases}
          permissionsLoading={permissionsLoading}
        />
      ) : null}
      {filteredCanvases.map((canvas) => (
        <CanvasCard
          key={canvas.id}
          canvas={canvas}
          organizationId={organizationId}
          onEdit={onEditCanvas}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ))}
    </div>
  );
}

interface CanvasCardProps {
  canvas: CanvasCardData;
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasCard({
  canvas,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardProps) {
  const canvasHref = `/${organizationId}/canvases/${canvas.id}`;
  const previewNodes = canvas.nodes || [];
  const previewEdges = canvas.edges || [];

  return (
    <div className="relative min-h-48 bg-white dark:bg-gray-800 rounded-md outline outline-gray-950/15 hover:shadow-md transition-shadow cursor-pointer">
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex flex-col h-full">
        <div className="p-4">
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col flex-1 min-w-0">
              <Heading
                level={3}
                className="!text-base font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate"
              >
                <span className="truncate">{canvas.name}</span>
              </Heading>
            </div>
            <div className="pointer-events-auto">
              <CanvasActionsMenu
                canvas={canvas}
                organizationId={organizationId}
                onEdit={onEdit}
                canUpdateCanvases={canUpdateCanvases}
                canDeleteCanvases={canDeleteCanvases}
                permissionsLoading={permissionsLoading}
              />
            </div>
          </div>

          {canvas.description ? (
            <div className="mb-4">
              <Text className="text-[13px] !leading-normal text-left text-gray-800 dark:text-gray-400 line-clamp-3">
                {canvas.description}
              </Text>
            </div>
          ) : null}

          <div className="flex justify-between items-center">
            <p className="text-xs text-gray-500 dark:text-gray-400 leading-none text-left mt-1">
              {canvas.createdBy?.name ? (
                <>
                  Created by {canvas.createdBy.name}, on {canvas.createdAt}
                </>
              ) : (
                <>Created on {canvas.createdAt}</>
              )}
            </p>
          </div>
        </div>

        <CanvasMiniMap nodes={previewNodes} edges={previewEdges} />
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
      <div className="p-4">
        <div className="h-28 w-full bg-transparent flex flex-col items-center justify-center gap-1 text-[13px] text-gray-500">
          <Rainbow size={24} className="text-gray-500" />
          Canvas is empty
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
    <div className="p-4 w-full overflow-hidden">
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

interface CanvasActionsMenuProps {
  canvas: CanvasCardData;
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasActionsMenu({
  canvas,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const deleteCanvasMutation = useDeleteCanvas(organizationId);
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const canManage = canUpdateCanvases || canDeleteCanvases;

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleDelete = async () => {
    if (!canDeleteCanvases) return;
    // If we're currently viewing this workflow, navigate immediately and remove from cache to prevent 404
    const currentPath = location.pathname;
    const canvasPath = `/${organizationId}/canvases/${canvas.id}`;
    const isViewingCanvas = currentPath === canvasPath || currentPath.startsWith(`${canvasPath}/`);

    if (isViewingCanvas) {
      // Remove from cache FIRST to prevent any queries from running
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvas.id) });
      // Navigate immediately with replace to avoid back button issues and prevent 404 flash
      navigate(`/${organizationId}`, { replace: true });
      // Then delete (fire and forget)
      deleteCanvasMutation.mutate(canvas.id, {
        onSuccess: () => {
          showSuccessToast("Canvas deleted successfully");
          closeDialog();
        },
        onError: (_error) => {
          showErrorToast("Failed to delete canvas");
        },
      });
      return;
    }

    try {
      await deleteCanvasMutation.mutateAsync(canvas.id);
      showSuccessToast("Canvas deleted successfully");
      closeDialog();
    } catch {
      showErrorToast("Failed to delete canvas");
    }
  };

  return (
    <>
      <div
        className="flex-shrink-0"
        onClick={(event: MouseEvent<HTMLDivElement>) => {
          event.preventDefault();
          event.stopPropagation();
        }}
      >
        {!canManage ? (
          <PermissionTooltip
            allowed={canManage || permissionsLoading}
            message="You don't have permission to manage this canvas."
          >
            <button
              className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
              aria-label="Canvas actions"
              disabled
            >
              <MoreVertical size={16} />
            </button>
          </PermissionTooltip>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger
              asChild
              onClick={(event: MouseEvent<HTMLButtonElement>) => {
                event.preventDefault();
                event.stopPropagation();
              }}
            >
              <button
                className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
                aria-label="Canvas actions"
                disabled={deleteCanvasMutation.isPending}
              >
                <MoreVertical size={16} />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <PermissionTooltip
                allowed={canUpdateCanvases || permissionsLoading}
                message="You don't have permission to update canvases."
              >
                <DropdownMenuItem
                  onClick={(event: MouseEvent<HTMLElement>) => {
                    event.preventDefault();
                    event.stopPropagation();
                    if (!canUpdateCanvases) return;
                    onEdit(canvas);
                  }}
                  disabled={!canUpdateCanvases}
                >
                  <Pencil size={16} />
                  Change Name
                </DropdownMenuItem>
              </PermissionTooltip>
              <PermissionTooltip
                allowed={canDeleteCanvases || permissionsLoading}
                message="You don't have permission to delete canvases."
              >
                <DropdownMenuItem
                  onClick={openDialog}
                  className="text-red-600 dark:text-red-400 focus:text-red-600 dark:focus:text-red-400"
                  disabled={!canDeleteCanvases}
                >
                  <Trash2 size={16} />
                  Delete
                </DropdownMenuItem>
              </PermissionTooltip>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <Dialog open={isDialogOpen} onClose={closeDialog} size="lg" className="text-left">
        <DialogTitle className="text-gray-800 dark:text-red-100">Delete "{canvas.name}"?</DialogTitle>
        <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
          This cannot be undone. Are you sure you want to continue?
        </DialogDescription>
        <DialogActions>
          <LoadingButton
            variant="destructive"
            onClick={(event) => {
              event.stopPropagation();
              handleDelete();
            }}
            disabled={!canDeleteCanvases}
            loading={deleteCanvasMutation.isPending}
            loadingText="Deleting..."
            className="flex items-center gap-2"
          >
            <Trash2 size={16} />
            Delete
          </LoadingButton>
          <Button
            variant="outline"
            onClick={(event) => {
              event.stopPropagation();
              closeDialog();
            }}
          >
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

function CanvasListRow({
  canvas,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasCardProps) {
  const canvasHref = `/${organizationId}/canvases/${canvas.id}`;

  return (
    <div className="relative overflow-hidden rounded-md bg-white outline outline-gray-950/15 hover:shadow-md dark:bg-gray-800">
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex w-full min-w-0 flex-col justify-center p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <Heading
              level={3}
              className="!text-base font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 truncate"
            >
              <span className="truncate">{canvas.name}</span>
            </Heading>
          </div>
          <div className="pointer-events-auto">
            <CanvasActionsMenu
              canvas={canvas}
              organizationId={organizationId}
              onEdit={onEdit}
              canUpdateCanvases={canUpdateCanvases}
              canDeleteCanvases={canDeleteCanvases}
              permissionsLoading={permissionsLoading}
            />
          </div>
        </div>
        {canvas.description ? (
          <Text className="mt-1 text-left text-[13px] !leading-normal text-gray-800 dark:text-gray-400 line-clamp-2">
            {canvas.description}
          </Text>
        ) : null}
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {canvas.createdBy?.name ? (
            <>
              Created by {canvas.createdBy.name}, on {canvas.createdAt}
            </>
          ) : (
            <>Created on {canvas.createdAt}</>
          )}
        </p>
      </div>
    </div>
  );
}

export default HomePage;
