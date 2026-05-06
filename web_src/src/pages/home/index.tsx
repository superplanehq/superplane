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
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
  type MouseEvent,
  type MutableRefObject,
  type RefObject,
} from "react";
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
  CANVAS_FOLDER_COLORS,
  DEFAULT_CANVAS_FOLDER_COLOR,
  canvasKeys,
  useCanvasFolders,
  useCanvases,
  useCreateCanvasFolder,
  useDeleteCanvas,
  useDeleteCanvasFolder,
  useUpdateCanvasFolder,
  useUpdateCanvasFolderPosition,
  useUpdateCanvasFolderMembership,
  type CanvasFolderColor,
} from "../../hooks/useCanvasData";
import { cn } from "../../lib/utils";
import { showErrorToast, showSuccessToast } from "../../lib/toast";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useCreateCanvasModalState } from "./useCreateCanvasModalState";
import { getApiErrorMessage } from "../../lib/errors";
import type {
  CanvasesCanvas,
  CanvasesCanvasFolder,
  SuperplaneComponentsEdge,
  SuperplaneComponentsNode,
} from "@/api-client";

interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  type: "canvases";
  canvasFolderId?: string;
  createdBy?: { id?: string; name?: string };
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
}

interface CanvasFolderData {
  id: string;
  title: string;
  backgroundColor: CanvasFolderColor;
}

const FOLDER_COLOR_OPTIONS: Record<CanvasFolderColor, { label: string; backgroundClass: string; swatchClass: string }> =
  {
    color_1: { label: "blue", backgroundClass: "bg-blue-500", swatchClass: "bg-blue-500" },
    color_2: { label: "green", backgroundClass: "bg-green-600", swatchClass: "bg-green-600" },
    color_3: { label: "violet", backgroundClass: "bg-violet-500", swatchClass: "bg-violet-500" },
    color_4: { label: "yellow", backgroundClass: "bg-yellow-950", swatchClass: "bg-yellow-950" },
    color_5: { label: "slate", backgroundClass: "bg-slate-700", swatchClass: "bg-slate-700" },
    color_6: { label: "orange", backgroundClass: "bg-orange-500", swatchClass: "bg-orange-500" },
  };

const compareByName = <T extends { name: string }>(left: T, right: T) => left.name.localeCompare(right.name);

function asCanvasFolderColor(value?: string): CanvasFolderColor {
  return CANVAS_FOLDER_COLORS.includes(value as CanvasFolderColor)
    ? (value as CanvasFolderColor)
    : DEFAULT_CANVAS_FOLDER_COLOR;
}

function toCanvasCardData(canvas: CanvasesCanvas, formatDate: (value?: string) => string): CanvasCardData | null {
  const id = canvas.metadata?.id;
  const name = canvas.metadata?.name;
  if (!id || !name) {
    return null;
  }

  return {
    id,
    name,
    description: canvas.metadata?.description,
    createdAt: formatDate(canvas.metadata?.createdAt),
    type: "canvases",
    canvasFolderId: canvas.metadata?.canvasFolderId || undefined,
    createdBy: canvas.metadata?.createdBy,
    nodes: canvas.spec?.nodes || [],
    edges: canvas.spec?.edges || [],
  };
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
    data: canvasFoldersData = [],
    isLoading: canvasFoldersLoading,
    error: canvasFoldersApiError,
  } = useCanvasFolders(organizationId || "");

  const canvasError =
    canvasesApiError || canvasFoldersApiError ? "Failed to fetch canvases. Please try again later." : null;
  const canCreateCanvases = canAct("canvases", "create");
  const canUpdateCanvases = canAct("canvases", "update");
  const canDeleteCanvases = canAct("canvases", "delete");

  const formatDate = (value?: string) => {
    if (!value) return "Unknown";
    return new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
  };

  const canvases: CanvasCardData[] = (canvasesData || [])
    .map((canvas: CanvasesCanvas) => toCanvasCardData(canvas, formatDate))
    .filter((canvas): canvas is CanvasCardData => canvas !== null)
    .sort(compareByName);

  const canvasFolders: CanvasFolderData[] = (canvasFoldersData || [])
    .map((folder: CanvasesCanvasFolder) => ({
      id: folder.metadata?.id || "",
      title: folder.spec?.title || "",
      backgroundColor: asCanvasFolderColor(folder.spec?.backgroundColor),
    }))
    .filter((folder) => folder.id && folder.title);

  const filteredCanvases = canvases.filter((canvas) => {
    const normalizedQuery = searchQuery.toLowerCase();
    return (
      canvas.name.toLowerCase().includes(normalizedQuery) || canvas.description?.toLowerCase().includes(normalizedQuery)
    );
  });

  const isLoading = canvasesLoading || canvasFoldersLoading;

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
                canvasFolders={canvasFolders}
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
  canvasFolders,
  organizationId,
  searchQuery,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: {
  filteredCanvases: CanvasCardData[];
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  searchQuery: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}) {
  const folderedLayout = useMemo(() => {
    const folderIDs = new Set(canvasFolders.map((folder) => folder.id));
    const canvasesByFolderID = new Map<string, CanvasCardData[]>();
    const unfiledCanvases: CanvasCardData[] = [];

    for (const folder of canvasFolders) {
      canvasesByFolderID.set(folder.id, []);
    }

    for (const canvas of filteredCanvases) {
      if (canvas.canvasFolderId && folderIDs.has(canvas.canvasFolderId)) {
        canvasesByFolderID.get(canvas.canvasFolderId)?.push(canvas);
        continue;
      }

      unfiledCanvases.push(canvas);
    }

    const visibleFolders = searchQuery
      ? canvasFolders.filter((folder) => (canvasesByFolderID.get(folder.id) || []).length > 0)
      : canvasFolders;

    return { canvasesByFolderID, unfiledCanvases, visibleFolders };
  }, [canvasFolders, filteredCanvases, searchQuery]);

  if (filteredCanvases.length === 0 && (searchQuery || canvasFolders.length === 0)) {
    return searchQuery ? <CanvasesSearchEmptyState /> : <CanvasesEmptyState />;
  }

  if (folderedLayout.visibleFolders.length === 0 && folderedLayout.unfiledCanvases.length === 0) {
    return searchQuery ? <CanvasesSearchEmptyState /> : <CanvasesEmptyState />;
  }

  return (
    <div className="space-y-6">
      {folderedLayout.visibleFolders.map((folder) => (
        <CanvasFolderSection
          key={folder.id}
          folder={folder}
          canvases={folderedLayout.canvasesByFolderID.get(folder.id) || []}
          canvasFolders={canvasFolders}
          organizationId={organizationId}
          onEditCanvas={onEditCanvas}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
          canMoveUp={canvasFolders.findIndex((canvasFolder) => canvasFolder.id === folder.id) > 0}
          canMoveDown={
            canvasFolders.findIndex((canvasFolder) => canvasFolder.id === folder.id) < canvasFolders.length - 1
          }
        />
      ))}

      {folderedLayout.unfiledCanvases.length > 0 ? (
        <CanvasCardsGrid
          canvases={folderedLayout.unfiledCanvases}
          canvasFolders={canvasFolders}
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

interface CanvasFolderSectionProps {
  folder: CanvasFolderData;
  canvases: CanvasCardData[];
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
}

interface CanvasFolderTitleProps {
  folder: CanvasFolderData;
  canUpdateCanvases: boolean;
  renameInputRef: RefObject<HTMLInputElement | null>;
  isRenaming: boolean;
  draftTitle: string;
  isPending: boolean;
  onDraftTitleChange: (title: string) => void;
  onStartRenaming: () => void;
  onSubmitRename: () => void;
  onCancelRenaming: () => void;
  onFocusRenameInput: () => void;
  isSubmittingRenameRef: MutableRefObject<boolean>;
  ignoreBlurUntilRef: MutableRefObject<number>;
}

function CanvasFolderTitle({
  folder,
  canUpdateCanvases,
  renameInputRef,
  isRenaming,
  draftTitle,
  isPending,
  onDraftTitleChange,
  onStartRenaming,
  onSubmitRename,
  onCancelRenaming,
  onFocusRenameInput,
  isSubmittingRenameRef,
  ignoreBlurUntilRef,
}: CanvasFolderTitleProps) {
  const handleRenameKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      onSubmitRename();
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      onCancelRenaming();
    }
  };

  if (!canUpdateCanvases) {
    return (
      <Heading level={3} className="mb-0 truncate !text-base font-medium text-white">
        {folder.title}
      </Heading>
    );
  }

  if (isRenaming) {
    return (
      <Input
        ref={renameInputRef}
        value={draftTitle}
        onChange={(event) => onDraftTitleChange(event.target.value)}
        onBlur={() => {
          if (ignoreBlurUntilRef.current > Date.now()) {
            onFocusRenameInput();
            return;
          }

          if (!isSubmittingRenameRef.current) {
            onSubmitRename();
          }
        }}
        onKeyDown={handleRenameKeyDown}
        aria-label="Folder name"
        maxLength={128}
        disabled={isPending}
        className="h-6 max-w-[320px] border-white/50 bg-white/5 px-1 text-base font-medium text-white shadow-none placeholder:text-white/60 focus-visible:border-white/60"
      />
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={onStartRenaming}
          className="flex h-6 max-w-xl items-center rounded-md border border-transparent px-1 text-left transition hover:border-white/25 hover:bg-white/5"
          aria-label={`Rename folder ${folder.title}`}
        >
          <span className="truncate text-base font-medium text-white">{folder.title}</span>
        </button>
      </TooltipTrigger>
      <TooltipContent>Rename</TooltipContent>
    </Tooltip>
  );
}

function CanvasFolderSection({
  folder,
  canvases,
  canvasFolders,
  organizationId,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
}: CanvasFolderSectionProps) {
  const [draftTitle, setDraftTitle] = useState(folder.title);
  const [isRenaming, setIsRenaming] = useState(false);
  const renameInputRef = useRef<HTMLInputElement>(null);
  const isSubmittingRenameRef = useRef(false);
  const ignoreBlurUntilRef = useRef(0);
  const updateCanvasFolderMutation = useUpdateCanvasFolder(organizationId);

  useEffect(() => {
    if (!isRenaming) {
      setDraftTitle(folder.title);
    }
  }, [folder.title, isRenaming]);

  const focusRenameInput = (selectText = false) => {
    window.setTimeout(() => {
      renameInputRef.current?.focus();
      if (selectText) {
        renameInputRef.current?.select();
      }
    }, 0);
  };

  const startRenaming = ({ preserveFocus = false }: { preserveFocus?: boolean } = {}) => {
    if (!canUpdateCanvases || updateCanvasFolderMutation.isPending) return;

    if (preserveFocus) {
      ignoreBlurUntilRef.current = Date.now() + 200;
    }

    setIsRenaming(true);
    focusRenameInput(true);
  };

  const cancelRenaming = () => {
    setDraftTitle(folder.title);
    setIsRenaming(false);
  };

  const submitRename = async () => {
    if (!canUpdateCanvases || isSubmittingRenameRef.current) return;

    const title = draftTitle.trim();
    if (!title) {
      showErrorToast("Folder name is required");
      focusRenameInput();
      return;
    }

    if (title === folder.title) {
      cancelRenaming();
      return;
    }

    isSubmittingRenameRef.current = true;

    try {
      await updateCanvasFolderMutation.mutateAsync({
        folderId: folder.id,
        title,
        backgroundColor: folder.backgroundColor,
      });
      setIsRenaming(false);
      showSuccessToast("Folder renamed");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename folder"));
      focusRenameInput();
    } finally {
      isSubmittingRenameRef.current = false;
    }
  };

  return (
    <section className={cn("w-full rounded-md p-4", FOLDER_COLOR_OPTIONS[folder.backgroundColor].backgroundClass)}>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          <CanvasFolderTitle
            folder={folder}
            canUpdateCanvases={canUpdateCanvases}
            renameInputRef={renameInputRef}
            isRenaming={isRenaming}
            draftTitle={draftTitle}
            isPending={updateCanvasFolderMutation.isPending}
            onDraftTitleChange={setDraftTitle}
            onStartRenaming={() => startRenaming()}
            onSubmitRename={() => void submitRename()}
            onCancelRenaming={cancelRenaming}
            onFocusRenameInput={focusRenameInput}
            isSubmittingRenameRef={isSubmittingRenameRef}
            ignoreBlurUntilRef={ignoreBlurUntilRef}
          />
        </div>
        <CanvasFolderActionsMenu
          folder={folder}
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
          canvasFolders={canvasFolders}
          organizationId={organizationId}
          onEditCanvas={onEditCanvas}
          canUpdateCanvases={canUpdateCanvases}
          canDeleteCanvases={canDeleteCanvases}
          permissionsLoading={permissionsLoading}
        />
      ) : (
        <div className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-md px-4 py-8 text-center text-[13px] font-medium text-white/80">
          <FolderOpen size={18} className="text-white/80" />
          <span>No canvases in this folder</span>
        </div>
      )}
    </section>
  );
}

function CanvasCardsGrid({
  canvases,
  canvasFolders,
  organizationId,
  onEditCanvas,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: {
  canvases: CanvasCardData[];
  canvasFolders: CanvasFolderData[];
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
          canvasFolders={canvasFolders}
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
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasCard({
  canvas,
  canvasFolders,
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
                canvasFolders={canvasFolders}
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

interface CanvasFolderActionsMenuProps {
  folder: CanvasFolderData;
  organizationId: string;
  canUpdateCanvases: boolean;
  permissionsLoading: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onRenameRequest: () => void;
}

interface CanvasFolderMenuContentProps {
  folder: CanvasFolderData;
  canMoveUp: boolean;
  canMoveDown: boolean;
  isUpdatingFolder: boolean;
  isMovingFolder: boolean;
  isDeletingFolder: boolean;
  shouldStartRename: boolean;
  onMove: (direction: "DIRECTION_UP" | "DIRECTION_DOWN") => void;
  onRenameRequest: () => void;
  onColorChange: (backgroundColor: CanvasFolderColor) => void;
  onOpenDeleteDialog: () => void;
}

function CanvasFolderMenuContent({
  folder,
  canMoveUp,
  canMoveDown,
  isUpdatingFolder,
  isMovingFolder,
  isDeletingFolder,
  shouldStartRename,
  onMove,
  onRenameRequest,
  onColorChange,
  onOpenDeleteDialog,
}: CanvasFolderMenuContentProps) {
  const isMutating = isUpdatingFolder || isMovingFolder || isDeletingFolder;

  return (
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
          onMove("DIRECTION_UP");
        }}
        disabled={!canMoveUp || isMutating}
      >
        <ArrowUp size={16} />
        Move Up
      </DropdownMenuItem>
      <DropdownMenuItem
        onSelect={(event) => {
          event.preventDefault();
          onMove("DIRECTION_DOWN");
        }}
        disabled={!canMoveDown || isMutating}
      >
        <ArrowDown size={16} />
        Move Down
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        onSelect={(event) => {
          event.preventDefault();
          onRenameRequest();
        }}
        disabled={isMutating}
      >
        <Pencil size={16} />
        Change folder name
      </DropdownMenuItem>
      <DropdownMenuSub>
        <DropdownMenuSubTrigger>
          <Palette size={16} />
          Background
        </DropdownMenuSubTrigger>
        <DropdownMenuSubContent className="w-auto">
          <div className="flex items-center gap-2 p-2">
            {CANVAS_FOLDER_COLORS.map((color) => (
              <button
                key={color}
                type="button"
                aria-label={`${FOLDER_COLOR_OPTIONS[color].label} folder color`}
                className={cn(
                  "flex h-6 w-6 items-center justify-center rounded-full border border-slate-950/15 text-white",
                  FOLDER_COLOR_OPTIONS[color].swatchClass,
                  folder.backgroundColor === color && "ring-2 ring-gray-900 ring-offset-1",
                )}
                onClick={() => onColorChange(color)}
                disabled={color === folder.backgroundColor || isUpdatingFolder || isMovingFolder}
              >
                {folder.backgroundColor === color ? <Check className="h-3 w-3" /> : null}
              </button>
            ))}
          </div>
        </DropdownMenuSubContent>
      </DropdownMenuSub>
      <DropdownMenuItem
        onSelect={(event) => {
          event.preventDefault();
          onOpenDeleteDialog();
        }}
        disabled={isDeletingFolder}
      >
        <Trash2 size={16} />
        Remove Folder
      </DropdownMenuItem>
    </DropdownMenuContent>
  );
}

function CanvasFolderDeleteDialog({
  folder,
  open,
  canUpdateCanvases,
  isDeleting,
  onClose,
  onDelete,
}: {
  folder: CanvasFolderData;
  open: boolean;
  canUpdateCanvases: boolean;
  isDeleting: boolean;
  onClose: () => void;
  onDelete: () => void;
}) {
  return (
    <Dialog open={open} onClose={onClose} size="lg" className="text-left">
      <DialogTitle className="text-gray-800 dark:text-white">Remove "{folder.title}"?</DialogTitle>
      <DialogDescription className="text-sm text-gray-800 dark:text-gray-400">
        This will remove only the folder. The canvases will remain available.
      </DialogDescription>
      <DialogActions>
        <LoadingButton
          variant="default"
          onClick={onDelete}
          disabled={!canUpdateCanvases}
          loading={isDeleting}
          loadingText="Removing..."
          className="flex items-center gap-2"
        >
          <Trash2 size={16} />
          Remove Folder
        </LoadingButton>
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function DisabledCanvasFolderActionsButton({ allowed }: { allowed: boolean }) {
  return (
    <PermissionTooltip allowed={allowed} message="You don't have permission to update canvases.">
      <button
        className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
        aria-label="Folder actions"
        disabled
      >
        <MoreVertical size={16} />
      </button>
    </PermissionTooltip>
  );
}

function CanvasFolderActionsMenu({
  folder,
  organizationId,
  canUpdateCanvases,
  permissionsLoading,
  canMoveUp,
  canMoveDown,
  onRenameRequest,
}: CanvasFolderActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [shouldStartRename, setShouldStartRename] = useState(false);
  const [shouldOpenDeleteDialog, setShouldOpenDeleteDialog] = useState(false);
  const updateCanvasFolderMutation = useUpdateCanvasFolder(organizationId);
  const updateCanvasFolderPositionMutation = useUpdateCanvasFolderPosition(organizationId);
  const deleteCanvasFolderMutation = useDeleteCanvasFolder(organizationId);
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

  const handleColorChange = async (backgroundColor: CanvasFolderColor) => {
    if (!canUpdateCanvases || backgroundColor === folder.backgroundColor) return;

    try {
      await updateCanvasFolderMutation.mutateAsync({
        folderId: folder.id,
        title: folder.title,
        backgroundColor,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update folder color"));
    }
  };

  const handleDelete = async () => {
    if (!canUpdateCanvases) return;

    try {
      await deleteCanvasFolderMutation.mutateAsync(folder.id);
      showSuccessToast("Folder removed");
      setIsDialogOpen(false);
    } catch {
      showErrorToast("Failed to remove folder");
    }
  };

  const handleRenameRequest = () => {
    setShouldStartRename(true);
    setIsMenuOpen(false);
  };

  const handleMove = async (direction: "DIRECTION_UP" | "DIRECTION_DOWN") => {
    if (!canUpdateCanvases) return;

    try {
      await updateCanvasFolderPositionMutation.mutateAsync({
        folderId: folder.id,
        direction,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to move folder"));
    }
  };

  return (
    <>
      {!canUpdateCanvases ? (
        <DisabledCanvasFolderActionsButton allowed={allowed} />
      ) : (
        <DropdownMenu open={isMenuOpen} onOpenChange={setIsMenuOpen}>
          <DropdownMenuTrigger asChild>
            <button
              className="rounded p-1 text-white/80 hover:bg-white/10 disabled:cursor-not-allowed disabled:opacity-50"
              aria-label="Folder actions"
              disabled={
                updateCanvasFolderMutation.isPending ||
                updateCanvasFolderPositionMutation.isPending ||
                deleteCanvasFolderMutation.isPending
              }
            >
              <MoreVertical size={16} />
            </button>
          </DropdownMenuTrigger>
          <CanvasFolderMenuContent
            folder={folder}
            canMoveUp={canMoveUp}
            canMoveDown={canMoveDown}
            isUpdatingFolder={updateCanvasFolderMutation.isPending}
            isMovingFolder={updateCanvasFolderPositionMutation.isPending}
            isDeletingFolder={deleteCanvasFolderMutation.isPending}
            shouldStartRename={shouldStartRename}
            onMove={(direction) => void handleMove(direction)}
            onRenameRequest={handleRenameRequest}
            onColorChange={(color) => void handleColorChange(color)}
            onOpenDeleteDialog={() => {
              setShouldOpenDeleteDialog(true);
              setIsMenuOpen(false);
            }}
          />
        </DropdownMenu>
      )}

      <CanvasFolderDeleteDialog
        folder={folder}
        open={isDialogOpen}
        canUpdateCanvases={canUpdateCanvases}
        isDeleting={deleteCanvasFolderMutation.isPending}
        onClose={() => setIsDialogOpen(false)}
        onDelete={() => void handleDelete()}
      />
    </>
  );
}

interface CanvasActionsMenuProps {
  canvas: CanvasCardData;
  canvasFolders: CanvasFolderData[];
  organizationId: string;
  onEdit: (canvas: CanvasCardData) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}

function CanvasActionsMenu({
  canvas,
  canvasFolders,
  organizationId,
  onEdit,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: CanvasActionsMenuProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [newFolderTitle, setNewFolderTitle] = useState("");
  const [newFolderColor, setNewFolderColor] = useState<CanvasFolderColor>(DEFAULT_CANVAS_FOLDER_COLOR);
  const deleteCanvasMutation = useDeleteCanvas(organizationId);
  const createCanvasFolderMutation = useCreateCanvasFolder(organizationId);
  const updateCanvasFolderMembershipMutation = useUpdateCanvasFolderMembership(organizationId);
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const canManage = canUpdateCanvases || canDeleteCanvases;
  const folderActionLabel = canvas.canvasFolderId ? "Move to Folder" : "Add to Folder";
  const normalizedNewFolderTitle = newFolderTitle.trim().toLowerCase();
  const isDuplicateNewFolderTitle = canvasFolders.some(
    (folder) => folder.title.trim().toLowerCase() === normalizedNewFolderTitle,
  );

  const closeDialog = () => {
    setIsDialogOpen(false);
  };

  const openDialog = (event: MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDialogOpen(true);
  };

  const handleAssignToFolder = async (folderId: string) => {
    if (!canUpdateCanvases || folderId === canvas.canvasFolderId) return;

    try {
      await updateCanvasFolderMembershipMutation.mutateAsync({ canvasId: canvas.id, folderId });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to add canvas to folder"));
    }
  };

  const handleRemoveFromFolder = async () => {
    if (!canUpdateCanvases || !canvas.canvasFolderId) return;

    try {
      await updateCanvasFolderMembershipMutation.mutateAsync({ canvasId: canvas.id });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to remove canvas from folder"));
    }
  };

  const handleCreateFolder = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!canUpdateCanvases) return;

    const title = newFolderTitle.trim();
    if (!title) return;

    if (isDuplicateNewFolderTitle) {
      showErrorToast("Folder name already exists");
      return;
    }

    try {
      const response = await createCanvasFolderMutation.mutateAsync({ title, backgroundColor: newFolderColor });
      let folderId = response.data?.folder?.metadata?.id;

      if (!folderId) {
        await queryClient.invalidateQueries({ queryKey: canvasKeys.folderList(organizationId) });
        await queryClient.refetchQueries({ queryKey: canvasKeys.folderList(organizationId), type: "active" });

        const folders = queryClient.getQueryData<CanvasesCanvasFolder[]>(canvasKeys.folderList(organizationId)) || [];
        folderId =
          folders.find((folder) => folder.spec?.title?.trim().toLowerCase() === title.toLowerCase())?.metadata?.id ||
          "";
      }

      if (!folderId) {
        throw new Error("missing canvas folder id");
      }

      await updateCanvasFolderMembershipMutation.mutateAsync({ canvasId: canvas.id, folderId });

      setNewFolderTitle("");
      setNewFolderColor(DEFAULT_CANVAS_FOLDER_COLOR);
      showSuccessToast("Folder created");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to create folder"));
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
                disabled={deleteCanvasMutation.isPending || updateCanvasFolderMembershipMutation.isPending}
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
                  {folderActionLabel}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent className="w-64">
                  {canvasFolders.length > 0 && (
                    <>
                      {canvasFolders.map((folder) => (
                        <DropdownMenuItem
                          key={folder.id}
                          onClick={() => handleAssignToFolder(folder.id)}
                          disabled={
                            folder.id === canvas.canvasFolderId || updateCanvasFolderMembershipMutation.isPending
                          }
                        >
                          <span
                            className={cn(
                              "h-3 w-3 rounded-full",
                              FOLDER_COLOR_OPTIONS[folder.backgroundColor].swatchClass,
                            )}
                          />
                          <span className="truncate">{folder.title}</span>
                          {folder.id === canvas.canvasFolderId ? <Check className="ml-auto h-4 w-4" /> : null}
                        </DropdownMenuItem>
                      ))}

                      <DropdownMenuSeparator />
                    </>
                  )}

                  <form
                    className="space-y-3 p-3"
                    onSubmit={handleCreateFolder}
                    onClick={(event) => event.stopPropagation()}
                  >
                    <Input
                      value={newFolderTitle}
                      onChange={(event) => setNewFolderTitle(event.target.value)}
                      onKeyDown={(event) => event.stopPropagation()}
                      placeholder="New folder name"
                      className="h-8"
                      maxLength={128}
                      disabled={!canUpdateCanvases || createCanvasFolderMutation.isPending}
                    />
                    {isDuplicateNewFolderTitle ? (
                      <Text className="text-xs text-red-600 dark:text-red-300">Folder name already exists</Text>
                    ) : null}
                    <div className="flex items-center gap-2">
                      {CANVAS_FOLDER_COLORS.map((color) => (
                        <button
                          key={color}
                          type="button"
                          aria-label={`${FOLDER_COLOR_OPTIONS[color].label} folder color`}
                          className={cn(
                            "flex h-5 w-5 items-center justify-center rounded-full border border-slate-950/15 text-white",
                            FOLDER_COLOR_OPTIONS[color].swatchClass,
                            newFolderColor === color && "ring-2 ring-gray-900 ring-offset-1",
                          )}
                          onClick={() => setNewFolderColor(color)}
                        >
                          {newFolderColor === color ? <Check className="h-3 w-3" /> : null}
                        </button>
                      ))}
                    </div>
                    <Button
                      type="submit"
                      size="sm"
                      className="w-full"
                      disabled={
                        !newFolderTitle.trim() || isDuplicateNewFolderTitle || createCanvasFolderMutation.isPending
                      }
                    >
                      Create Folder
                    </Button>
                  </form>
                </DropdownMenuSubContent>
              </DropdownMenuSub>

              {canvas.canvasFolderId ? (
                <DropdownMenuItem
                  onClick={handleRemoveFromFolder}
                  disabled={!canUpdateCanvases || updateCanvasFolderMembershipMutation.isPending}
                >
                  <FolderMinus size={16} />
                  Remove from Folder
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
