import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { Palette, Plus, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { CreateCanvasModal } from "../../components/CreateCanvasModal";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/AccountContext";
import { usePermissions } from "@/contexts/PermissionsContext";
import { PermissionTooltip } from "@/components/PermissionGate";
import {
  CANVAS_FOLDER_COLORS,
  DEFAULT_CANVAS_FOLDER_COLOR,
  useCanvasFolders,
  useCanvases,
  type CanvasFolderColor,
} from "../../hooks/useCanvasData";
import { Button } from "@/components/ui/button";
import { useCreateCanvasModalState } from "./useCreateCanvasModalState";
import type { CanvasFoldersCanvasFolder, CanvasesCanvas } from "@/api-client";
import { CanvasCardsGrid } from "./CanvasCardsGrid";
import { CanvasFolderSection } from "./CanvasFolderSection";
import type { CanvasCardData, CanvasFolderData } from "./types";

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
    canvasFolderId: canvas.metadata?.folderId || undefined,
    createdBy: canvas.metadata?.createdBy,
    nodes: canvas.spec?.nodes || [],
    edges: canvas.spec?.edges || [],
  };
}

export function HomePage() {
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
    .map((folder: CanvasFoldersCanvasFolder) => ({
      id: folder.metadata?.id || "",
      title: folder.spec?.title || "",
      backgroundColor: asCanvasFolderColor(folder.spec?.backgroundColor),
      canvasIds: folder.spec?.canvases?.map((canvas) => canvas.id || "").filter(Boolean) || [],
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
}

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
