import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePageTitle } from "@/hooks/usePageTitle";
import {
  ArrowDown,
  ArrowUp,
  Check,
  FolderMinus,
  FolderOpen,
  FolderPlus,
  MoreVertical,
  Palette,
  Pencil,
  Plus,
  Search,
  Trash2,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState, type FormEvent, type KeyboardEvent, type MouseEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { CreateCanvasModal } from "../../components/CreateCanvasModal";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "../../components/Dialog/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "../../ui/dropdownMenu";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/AccountContext";
import { usePermissions } from "@/contexts/PermissionsContext";
import { PermissionTooltip } from "@/components/PermissionGate";
import {
  CANVAS_GROUP_COLORS,
  canvasKeys,
  useCanvasGroups,
  useCanvases,
  useCreateCanvasGroup,
  useDeleteCanvas,
  useDeleteCanvasGroup,
  useUpdateCanvasGroup,
  useUpdateCanvasGroupPosition,
  useUpdateCanvasGroupMembership,
  type CanvasGroupColor,
} from "../../hooks/useCanvasData";
import { cn } from "../../lib/utils";
import { showErrorToast, showSuccessToast } from "../../lib/toast";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useCreateCanvasModalState } from "./useCreateCanvasModalState";
import { getApiErrorMessage } from "../../lib/errors";
import type {
  CanvasesCanvas,
  CanvasesCanvasGroup,
  SuperplaneComponentsEdge,
  SuperplaneComponentsNode,
} from "@/api-client";

interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "canvases";
  canvasGroupId?: string;
  createdBy?: { id?: string; name?: string };
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
}

interface CanvasGroupData {
  id: string;
  title: string;
  backgroundColor: CanvasGroupColor;
}

const GROUP_BACKGROUND_CLASSES: Record<CanvasGroupColor, string> = {
  "blue-800": "bg-blue-800",
  "green-800": "bg-green-800",
  "slate-700": "bg-slate-700",
  "violet-800": "bg-violet-800",
  "yellow-800": "bg-yellow-800",
};

const GROUP_SWATCH_CLASSES: Record<CanvasGroupColor, string> = {
  "blue-800": "bg-blue-800",
  "green-800": "bg-green-800",
  "slate-700": "bg-slate-700",
  "violet-800": "bg-violet-800",
  "yellow-800": "bg-yellow-800",
};

const compareByName = <T extends { name: string }>(left: T, right: T) => left.name.localeCompare(right.name);
function asCanvasGroupColor(value?: string): CanvasGroupColor {
  return CANVAS_GROUP_COLORS.includes(value as CanvasGroupColor) ? (value as CanvasGroupColor) : "blue-800";
}

const HomePage = () => {
  usePageTitle(["Home"]);

  const [searchQuery, setSearchQuery] = useState("");
  const canvasModalState = useCreateCanvasModalState();

  const { organizationId } = useParams<{ organizationId: string }>();
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const {
    data: canvasesData = [],
    isLoading: canvasesLoading,
    error: canvasesApiError,
  } = useCanvases(organizationId || "");
  const {
    data: canvasGroupsData = [],
    isLoading: canvasGroupsLoading,
    error: canvasGroupsApiError,
  } = useCanvasGroups(organizationId || "");

  const canvasError =
    canvasesApiError || canvasGroupsApiError ? "Failed to fetch canvases. Please try again later." : null;
  const canCreateCanvases = canAct("canvases", "create");
  const canUpdateCanvases = canAct("canvases", "update");
  const canDeleteCanvases = canAct("canvases", "delete");

  const formatDate = (value?: string) => {
    if (!value) return "Unknown";
    return new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  };

  const canvases: CanvasCardData[] = (canvasesData || [])
    .map((canvas: CanvasesCanvas) => ({
      id: canvas.metadata?.id!,
      name: canvas.metadata?.name!,
      description: canvas.metadata?.description,
      createdAt: formatDate(canvas.metadata?.createdAt),
      type: "canvases" as const,
      canvasGroupId: canvas.metadata?.canvasGroupId || undefined,
      createdBy: canvas.metadata?.createdBy,
      nodes: canvas.spec?.nodes || [],
      edges: canvas.spec?.edges || [],
    }))
    .sort(compareByName);

  const canvasGroups: CanvasGroupData[] = (canvasGroupsData || [])
    .map((group: CanvasesCanvasGroup) => ({
      id: group.metadata?.id || "",
      title: group.spec?.title || "",
      backgroundColor: asCanvasGroupColor(group.spec?.backgroundColor),
    }))
    .filter((group) => group.id && group.title);

  const filteredCanvases = canvases.filter((canvas) => {
    const normalizedQuery = searchQuery.toLowerCase();
    return (
      canvas.name.toLowerCase().includes(normalizedQuery) || canvas.description?.toLowerCase().includes(normalizedQuery)
    );
  });

  const isLoading = canvasesLoading || canvasGroupsLoading;

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

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-slate-900">
      <header className="bg-white border-b border-slate-950/15 px-4 h-12 flex items-center">
        <OrganizationMenuButton organizationId={organizationId} />
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="bg-slate-100 w-full flex-grow-1">
          <div className="mx-auto w-full max-w-6xl p-8">
            <div className="mb-6 flex items-center justify-between">
              <div>
                <Heading level={2} className="!text-2xl mb-1">
                  Canvases
                </Heading>
                <Text className="text-gray-800 dark:text-gray-400">
                  Overview of all mapped automations across your organization.
                </Text>
              </div>
            </div>

            <div className="mb-6">
              <CanvasToolbar
                organizationId={organizationId}
                searchQuery={searchQuery}
                setSearchQuery={setSearchQuery}
                canCreateCanvases={canCreateCanvases}
                permissionsLoading={permissionsLoading}
              />
            </div>

            {canvasError ? (
              <div className="bg-white border border-red-300 text-red-500 px-4 py-2 rounded">
                <Text>{canvasError}</Text>
              </div>
            ) : (
              <Content
                filteredCanvases={filteredCanvases}
                canvasGroups={canvasGroups}
                organizationId={organizationId}
                searchQuery={searchQuery}
                onEditCanvas={canvasModalState.onOpenEdit}
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

interface CanvasToolbarProps {
  organizationId: string;
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  canCreateCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasToolbar({
  organizationId,
  searchQuery,
  setSearchQuery,
  canCreateCanvases,
  permissionsLoading,
}: CanvasToolbarProps) {
  const allowed = canCreateCanvases || permissionsLoading;

  return (
    <div className="flex w-full flex-col gap-3 sm:flex-row sm:items-center">
      <PermissionTooltip allowed={allowed} message="You don't have permission to create canvases.">
        {allowed ? (
          <Button asChild>
            <Link to={`/${organizationId}/canvases/new`} aria-label="Create new canvas">
              <Plus className="h-4 w-4" />
              New Canvas
            </Link>
          </Button>
        ) : (
          <Button type="button" disabled>
            <Plus className="h-4 w-4" />
            New Canvas
          </Button>
        )}
      </PermissionTooltip>

      <div className="min-w-0 w-full sm:ml-auto sm:w-80">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
          <Input
            placeholder="Filter canvases..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
      </div>
    </div>
  );
}

function Content({
  filteredCanvases,
  canvasGroups,
  organizationId,
  searchQuery,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: {
  filteredCanvases: CanvasCardData[];
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  searchQuery: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}) {
  const groupedLayout = useMemo(() => {
    const groupIDs = new Set(canvasGroups.map((group) => group.id));
    const canvasesByGroupID = new Map<string, CanvasCardData[]>();
    const ungroupedCanvases: CanvasCardData[] = [];

    for (const group of canvasGroups) {
      canvasesByGroupID.set(group.id, []);
    }

    for (const canvas of filteredCanvases) {
      if (canvas.canvasGroupId && groupIDs.has(canvas.canvasGroupId)) {
        canvasesByGroupID.get(canvas.canvasGroupId)?.push(canvas);
        continue;
      }

      ungroupedCanvases.push(canvas);
    }

    const visibleGroups = searchQuery
      ? canvasGroups.filter((group) => (canvasesByGroupID.get(group.id) || []).length > 0)
      : canvasGroups;

    return { canvasesByGroupID, ungroupedCanvases, visibleGroups };
  }, [canvasGroups, filteredCanvases, searchQuery]);

  if (filteredCanvases.length === 0 && (searchQuery || canvasGroups.length === 0)) {
    return searchQuery ? <CanvasesSearchEmptyState /> : <CanvasesEmptyState />;
  }

  if (groupedLayout.visibleGroups.length === 0 && groupedLayout.ungroupedCanvases.length === 0) {
    return searchQuery ? <CanvasesSearchEmptyState /> : <CanvasesEmptyState />;
  }

  return (
    <div className="space-y-6">
      {groupedLayout.visibleGroups.map((group) => (
        <CanvasGroupSection
          key={group.id}
          group={group}
          canvases={groupedLayout.canvasesByGroupID.get(group.id) || []}
          canvasGroups={canvasGroups}
          organizationId={organizationId}
          onEditCanvas={onEditCanvas}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
          canMoveUp={canvasGroups.findIndex((canvasGroup) => canvasGroup.id === group.id) > 0}
          canMoveDown={canvasGroups.findIndex((canvasGroup) => canvasGroup.id === group.id) < canvasGroups.length - 1}
        />
      ))}

      {groupedLayout.ungroupedCanvases.length > 0 ? (
        <CanvasCardsGrid
          canvases={groupedLayout.ungroupedCanvases}
          canvasGroups={canvasGroups}
          organizationId={organizationId}
          onEditCanvas={onEditCanvas}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ) : null}
    </div>
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

function CanvasesEmptyState() {
  return (
    <div className="text-center py-12">
      <Palette className="mx-auto text-gray-400 mb-4" size={48} aria-hidden />
      <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
        No canvases yet
      </Heading>
    </div>
  );
}

interface CanvasGroupSectionProps {
  group: CanvasGroupData;
  canvases: CanvasCardData[];
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
}

function CanvasGroupSection({
  group,
  canvases,
  canvasGroups,
  organizationId,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
}: CanvasGroupSectionProps) {
  const [draftTitle, setDraftTitle] = useState(group.title);
  const [isRenaming, setIsRenaming] = useState(false);
  const renameInputRef = useRef<HTMLInputElement>(null);
  const isSubmittingRenameRef = useRef(false);
  const ignoreBlurUntilRef = useRef(0);
  const updateCanvasGroupMutation = useUpdateCanvasGroup(organizationId);

  useEffect(() => {
    if (!isRenaming) {
      setDraftTitle(group.title);
    }
  }, [group.title, isRenaming]);

  const focusRenameInput = (selectText = false) => {
    window.setTimeout(() => {
      renameInputRef.current?.focus();
      if (selectText) {
        renameInputRef.current?.select();
      }
    }, 0);
  };

  const startRenaming = ({ preserveFocus = false }: { preserveFocus?: boolean } = {}) => {
    if (!canUpdateCanvases || updateCanvasGroupMutation.isPending) return;

    if (preserveFocus) {
      ignoreBlurUntilRef.current = Date.now() + 200;
    }

    setIsRenaming(true);
    focusRenameInput(true);
  };

  const cancelRenaming = () => {
    setDraftTitle(group.title);
    setIsRenaming(false);
  };

  const submitRename = async () => {
    if (!canUpdateCanvases || isSubmittingRenameRef.current) return;

    const title = draftTitle.trim();
    if (!title) {
      showErrorToast("Group name is required");
      focusRenameInput();
      return;
    }

    if (title === group.title) {
      cancelRenaming();
      return;
    }

    isSubmittingRenameRef.current = true;

    try {
      await updateCanvasGroupMutation.mutateAsync({
        groupId: group.id,
        title,
        backgroundColor: group.backgroundColor,
      });
      setIsRenaming(false);
      showSuccessToast("Group renamed");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename group"));
      focusRenameInput();
    } finally {
      isSubmittingRenameRef.current = false;
    }
  };

  const handleRenameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      void submitRename();
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      cancelRenaming();
    }
  };

  return (
    <section className={cn("w-full rounded-md p-4", GROUP_BACKGROUND_CLASSES[group.backgroundColor])}>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          {canUpdateCanvases ? (
            isRenaming ? (
              <Input
                ref={renameInputRef}
                value={draftTitle}
                onChange={(event) => setDraftTitle(event.target.value)}
                onBlur={() => {
                  if (ignoreBlurUntilRef.current > Date.now()) {
                    focusRenameInput();
                    return;
                  }

                  if (!isSubmittingRenameRef.current) {
                    void submitRename();
                  }
                }}
                onKeyDown={handleRenameKeyDown}
                aria-label="Group name"
                maxLength={128}
                disabled={updateCanvasGroupMutation.isPending}
                className="h-6 max-w-[320px] border-white/50 bg-white/5 px-1 text-base font-medium text-white shadow-none placeholder:text-white/60 focus-visible:border-white/60"
              />
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => startRenaming()}
                    className="flex h-6 max-w-xl items-center rounded-md border border-transparent px-1 text-left transition hover:border-white/25 hover:bg-white/5"
                    aria-label={`Rename group ${group.title}`}
                  >
                    <span className="truncate text-base font-medium text-white">{group.title}</span>
                  </button>
                </TooltipTrigger>
                <TooltipContent>Rename</TooltipContent>
              </Tooltip>
            )
          ) : (
            <Heading level={3} className="mb-0 truncate !text-base font-medium text-white">
              {group.title}
            </Heading>
          )}
        </div>
        <CanvasGroupActionsMenu
          group={group}
          organizationId={organizationId}
          canUpdateCanvases={canUpdateCanvases}
          permissionsLoading={permissionsLoading}
          canMoveUp={canMoveUp}
          canMoveDown={canMoveDown}
          onRenameRequest={() => startRenaming({ preserveFocus: true })}
        />
      </div>

      {canvases.length > 0 ? (
        <CanvasCardsGrid
          canvases={canvases}
          canvasGroups={canvasGroups}
          organizationId={organizationId}
          onEditCanvas={onEditCanvas}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ) : (
        <div className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-md px-4 py-8 text-center text-[13px] font-medium text-white/80">
          <FolderOpen size={18} className="text-white/80" />
          <span>No canvases in this group</span>
        </div>
      )}
    </section>
  );
}

function CanvasCardsGrid({
  canvases,
  canvasGroups,
  organizationId,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: {
  canvases: CanvasCardData[];
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
      {canvases.map((canvas) => (
        <CanvasCard
          key={canvas.id}
          canvas={canvas}
          canvasGroups={canvasGroups}
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
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasCard({
  canvas,
  canvasGroups,
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
    <div className="relative min-h-40 bg-white dark:bg-gray-800 rounded-md outline outline-gray-950/15 hover:shadow-md transition-shadow cursor-pointer">
      <Link to={canvasHref} aria-label={`Open canvas ${canvas.name}`} className="absolute inset-0 rounded-md" />
      <div className="pointer-events-none relative flex flex-col h-full">
        <div className="p-3">
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col flex-1 min-w-0">
              <Heading
                level={3}
                className="mb-0 line-clamp-2 !text-sm font-medium text-gray-800 transition-colors !leading-5"
              >
                <span className="truncate">{canvas.name}</span>
              </Heading>
            </div>
            <div className="pointer-events-auto">
              <CanvasActionsMenu
                canvas={canvas}
                canvasGroups={canvasGroups}
                organizationId={organizationId}
                onEdit={onEdit}
                canUpdateCanvases={canUpdateCanvases}
                canDeleteCanvases={canDeleteCanvases}
                permissionsLoading={permissionsLoading}
              />
            </div>
          </div>

          {canvas.description ? (
            <div className="mb-3">
              <Text className="line-clamp-2 text-left text-[12px] !leading-normal text-gray-800 dark:text-gray-400">
                {canvas.description}
              </Text>
            </div>
          ) : null}

          <div className="flex justify-between items-center">
            <p className="mt-1 text-left text-[11px] leading-none text-gray-500 dark:text-gray-400">
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
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
}

function CanvasMiniMap({ nodes = [], edges = [] }: CanvasMiniMapProps) {
  const positionedNodes = nodes.filter(
    (node) => typeof node.position?.x === "number" && typeof node.position?.y === "number",
  ) as Array<SuperplaneComponentsNode & { position: { x: number; y: number } }>;

  if (!positionedNodes.length) {
    return <div className="h-24 w-full p-3 pt-0" />;
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
    <div className="w-full overflow-hidden p-3 pt-0">
      <svg
        viewBox={viewBox}
        preserveAspectRatio="xMidYMid meet"
        className="h-24 w-full text-gray-500 dark:text-gray-400"
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

interface CanvasGroupActionsMenuProps {
  group: CanvasGroupData;
  organizationId: string;
  canUpdateCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onRenameRequest: () => void;
}

function CanvasGroupActionsMenu({
  group,
  organizationId,
  canUpdateCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
  onRenameRequest,
}: CanvasGroupActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [shouldStartRename, setShouldStartRename] = useState(false);
  const [shouldOpenDeleteDialog, setShouldOpenDeleteDialog] = useState(false);
  const updateCanvasGroupMutation = useUpdateCanvasGroup(organizationId);
  const updateCanvasGroupPositionMutation = useUpdateCanvasGroupPosition(organizationId);
  const deleteCanvasGroupMutation = useDeleteCanvasGroup(organizationId);
  const allowed = canUpdateCanvases || permissionsLoading;

  useEffect(() => {
    if (!isMenuOpen && shouldStartRename) {
      setShouldStartRename(false);
      onRenameRequest();
    }
  }, [isMenuOpen, onRenameRequest, shouldStartRename]);

  useEffect(() => {
    if (!isMenuOpen && shouldOpenDeleteDialog) {
      setShouldOpenDeleteDialog(false);
      setIsDialogOpen(true);
    }
  }, [isMenuOpen, shouldOpenDeleteDialog]);

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const handleColorChange = async (backgroundColor: CanvasGroupColor) => {
    if (!canUpdateCanvases || backgroundColor === group.backgroundColor) return;

    try {
      await updateCanvasGroupMutation.mutateAsync({
        groupId: group.id,
        title: group.title,
        backgroundColor,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update group color"));
    }
  };

  const handleDelete = async () => {
    if (!canUpdateCanvases) return;

    try {
      await deleteCanvasGroupMutation.mutateAsync(group.id);
      showSuccessToast("Group removed");
      closeDialog();
    } catch {
      showErrorToast("Failed to remove group");
    }
  };

  const handleRenameRequest = () => {
    setShouldStartRename(true);
    setIsMenuOpen(false);
  };

  const handleMove = async (direction: "DIRECTION_UP" | "DIRECTION_DOWN") => {
    if (!canUpdateCanvases) return;

    try {
      await updateCanvasGroupPositionMutation.mutateAsync({
        groupId: group.id,
        direction,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to move group"));
    }
  };

  const handleOpenDeleteDialog = () => {
    setShouldOpenDeleteDialog(true);
    setIsMenuOpen(false);
  };

  return (
    <>
      {!canUpdateCanvases ? (
        <PermissionTooltip allowed={allowed} message="You don't have permission to update canvases.">
          <button
            className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="Group actions"
            disabled
          >
            <MoreVertical size={16} />
          </button>
        </PermissionTooltip>
      ) : (
        <DropdownMenu open={isMenuOpen} onOpenChange={setIsMenuOpen}>
          <DropdownMenuTrigger asChild>
            <button
              className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
              aria-label="Group actions"
              disabled={
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <MoreVertical size={16} />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            onCloseAutoFocus={(event) => {
              if (shouldStartRename) {
                event.preventDefault();
              }
            }}
          >
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                void handleMove("DIRECTION_UP");
              }}
              disabled={
                !canMoveUp ||
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <ArrowUp size={16} />
              Move Up
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                void handleMove("DIRECTION_DOWN");
              }}
              disabled={
                !canMoveDown ||
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <ArrowDown size={16} />
              Move Down
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                handleRenameRequest();
              }}
              disabled={
                updateCanvasGroupMutation.isPending ||
                updateCanvasGroupPositionMutation.isPending ||
                deleteCanvasGroupMutation.isPending
              }
            >
              <Pencil size={16} />
              Change group name
            </DropdownMenuItem>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Palette size={16} />
                Background
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="w-auto">
                <div className="flex items-center gap-2 p-2">
                  {CANVAS_GROUP_COLORS.map((color) => (
                    <button
                      key={color}
                      type="button"
                      aria-label={`${color.replace("-800", "")} group color`}
                      className={cn(
                        "flex h-6 w-6 items-center justify-center rounded-full border border-slate-950/15 text-white",
                        GROUP_SWATCH_CLASSES[color],
                        group.backgroundColor === color && "ring-2 ring-gray-900 ring-offset-1",
                      )}
                      onClick={() => void handleColorChange(color)}
                      disabled={
                        color === group.backgroundColor ||
                        updateCanvasGroupMutation.isPending ||
                        updateCanvasGroupPositionMutation.isPending
                      }
                    >
                      {group.backgroundColor === color ? <Check className="h-3 w-3" /> : null}
                    </button>
                  ))}
                </div>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                handleOpenDeleteDialog();
              }}
              disabled={deleteCanvasGroupMutation.isPending}
            >
              <Trash2 size={16} />
              Remove Group
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      <Dialog open={isDialogOpen} onClose={closeDialog} size="lg" className="text-left">
        <DialogTitle className="text-gray-800 dark:text-white">Remove "{group.title}"?</DialogTitle>
        <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
          This will remove only the folder. The canvases will remain available.
        </DialogDescription>
        <DialogActions>
          <LoadingButton
            variant="default"
            onClick={handleDelete}
            disabled={!canUpdateCanvases}
            loading={deleteCanvasGroupMutation.isPending}
            loadingText="Removing..."
            className="flex items-center gap-2"
          >
            <Trash2 size={16} />
            Remove Group
          </LoadingButton>
          <Button variant="outline" onClick={closeDialog}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

interface CanvasActionsMenuProps {
  canvas: CanvasCardData;
  canvasGroups: CanvasGroupData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasActionsMenu({
  canvas,
  canvasGroups,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [newGroupTitle, setNewGroupTitle] = useState("");
  const [newGroupColor, setNewGroupColor] = useState<CanvasGroupColor>("blue-800");
  const deleteCanvasMutation = useDeleteCanvas(organizationId);
  const createCanvasGroupMutation = useCreateCanvasGroup(organizationId);
  const updateCanvasGroupMembershipMutation = useUpdateCanvasGroupMembership(organizationId);
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

  const handleAssignToGroup = async (groupId: string) => {
    if (!canUpdateCanvases || groupId === canvas.canvasGroupId) return;

    try {
      await updateCanvasGroupMembershipMutation.mutateAsync({ canvasId: canvas.id, groupId });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add canvas to group"));
    }
  };

  const handleRemoveFromGroup = async () => {
    if (!canUpdateCanvases || !canvas.canvasGroupId) return;

    try {
      await updateCanvasGroupMembershipMutation.mutateAsync({ canvasId: canvas.id });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to remove canvas from group"));
    }
  };

  const handleCreateGroup = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canUpdateCanvases) return;

    const title = newGroupTitle.trim();
    if (!title) return;

    try {
      const response = await createCanvasGroupMutation.mutateAsync({ title, backgroundColor: newGroupColor });
      let groupId = response.data?.group?.metadata?.id;

      if (!groupId) {
        await queryClient.invalidateQueries({ queryKey: canvasKeys.groupList(organizationId) });
        await queryClient.refetchQueries({ queryKey: canvasKeys.groupList(organizationId), type: "active" });

        const groups = queryClient.getQueryData<CanvasesCanvasGroup[]>(canvasKeys.groupList(organizationId)) || [];
        groupId =
          groups.find((group) => group.spec?.title?.trim().toLowerCase() === title.toLowerCase())?.metadata?.id || "";
      }

      if (!groupId) {
        throw new Error("missing canvas group id");
      }

      await updateCanvasGroupMembershipMutation.mutateAsync({ canvasId: canvas.id, groupId });

      setNewGroupTitle("");
      setNewGroupColor("blue-800");
      showSuccessToast("Group created");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create group"));
    }
  };

  const handleDelete = async () => {
    if (!canDeleteCanvases) return;
    const currentPath = location.pathname;
    const canvasPath = `/${organizationId}/canvases/${canvas.id}`;
    const isViewingCanvas = currentPath === canvasPath || currentPath.startsWith(`${canvasPath}/`);

    if (isViewingCanvas) {
      queryClient.removeQueries({ queryKey: canvasKeys.detail(organizationId, canvas.id) });
      navigate(`/${organizationId}`, { replace: true });
      deleteCanvasMutation.mutate(canvas.id, {
        onSuccess: () => {
          showSuccessToast("Canvas deleted successfully");
          closeDialog();
        },
        onError: () => {
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
                disabled={deleteCanvasMutation.isPending || updateCanvasGroupMembershipMutation.isPending}
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

              <DropdownMenuSub>
                <DropdownMenuSubTrigger disabled={!canUpdateCanvases}>
                  <FolderPlus size={16} />
                  Add to Group
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent className="w-64">
                  {canvasGroups.length > 0 && (
                    <>
                      {canvasGroups.map((group) => (
                        <DropdownMenuItem
                          key={group.id}
                          onClick={() => handleAssignToGroup(group.id)}
                          disabled={group.id === canvas.canvasGroupId || updateCanvasGroupMembershipMutation.isPending}
                        >
                          <span className={cn("h-3 w-3 rounded-full", GROUP_SWATCH_CLASSES[group.backgroundColor])} />
                          <span className="truncate">{group.title}</span>
                          {group.id === canvas.canvasGroupId ? <Check className="ml-auto h-4 w-4" /> : null}
                        </DropdownMenuItem>
                      ))}

                      <DropdownMenuSeparator />
                    </>
                  )}

                  <form
                    className="space-y-3 p-3"
                    onSubmit={handleCreateGroup}
                    onClick={(event) => event.stopPropagation()}
                  >
                    <Input
                      value={newGroupTitle}
                      onChange={(event) => setNewGroupTitle(event.target.value)}
                      onKeyDown={(event) => event.stopPropagation()}
                      placeholder="New group name"
                      className="h-8"
                      maxLength={128}
                      disabled={!canUpdateCanvases || createCanvasGroupMutation.isPending}
                    />
                    <div className="flex items-center gap-2">
                      {CANVAS_GROUP_COLORS.map((color) => (
                        <button
                          key={color}
                          type="button"
                          aria-label={`${color.replace("-800", "")} group color`}
                          className={cn(
                            "flex h-5 w-5 items-center justify-center rounded-full border border-slate-950/15 text-white",
                            GROUP_SWATCH_CLASSES[color],
                            newGroupColor === color && "ring-2 ring-gray-900 ring-offset-1",
                          )}
                          onClick={() => setNewGroupColor(color)}
                        >
                          {newGroupColor === color ? <Check className="h-3 w-3" /> : null}
                        </button>
                      ))}
                    </div>
                    <Button
                      type="submit"
                      size="sm"
                      className="w-full"
                      disabled={!newGroupTitle.trim() || createCanvasGroupMutation.isPending}
                    >
                      Create Group
                    </Button>
                  </form>
                </DropdownMenuSubContent>
              </DropdownMenuSub>

              {canvas.canvasGroupId ? (
                <DropdownMenuItem
                  onClick={handleRemoveFromGroup}
                  disabled={!canUpdateCanvases || updateCanvasGroupMembershipMutation.isPending}
                >
                  <FolderMinus size={16} />
                  Remove from Group
                </DropdownMenuItem>
              ) : null}

              <PermissionTooltip
                allowed={canDeleteCanvases || permissionsLoading}
                message="You don't have permission to delete canvases."
              >
                <DropdownMenuItem onClick={openDialog} disabled={!canDeleteCanvases}>
                  <Trash2 size={16} />
                  Delete Canvas
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

export default HomePage;
