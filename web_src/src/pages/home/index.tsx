import { usePermissions } from "@/contexts/usePermissions";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { Palette } from "lucide-react";
import { useState } from "react";
import { Navigate, useParams } from "react-router-dom";
import { Heading } from "../../components/Heading/heading";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/useAccount";
import { CanvasCardsGrid } from "./CanvasCardsGrid";
import { CanvasFolderSection } from "./CanvasFolderSection";
import { CanvasToolbar } from "./CanvasToolbar";
import { EditAppModal } from "./EditAppModal";
import { HomePageShell } from "./HomePageShell";
import { CANVAS_FOLDER_SECTION_SHELL_CLASS } from "./canvasFolderStyles";
import type { CanvasCardData, CanvasFolderData } from "./types";
import { useEditApp } from "./useEditApp";
import { useHomePageCanvasList } from "./useHomePageCanvasList";

export function HomePage() {
  usePageTitle(["Home"]);

  const [searchQuery, setSearchQuery] = useState("");

  const { organizationId } = useParams<{ organizationId: string }>();
  const { account } = useAccount();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const {
    editingCanvas,
    openEdit,
    closeEdit,
    saveApp,
    isSaving: isEditAppSaving,
    isOpen: isEditAppModalOpen,
  } = useEditApp();

  const { canvases, canvasFolders, filteredCanvases, isLoading, isFetching, canvasError } = useHomePageCanvasList(
    organizationId,
    searchQuery,
  );
  const canUpdateCanvases = canAct("canvases", "update");
  const canDeleteCanvases = canAct("canvases", "delete");

  const isHomePageLoading = isLoading || (isFetching && canvases.length === 0 && canvasFolders.length === 0);
  useReportPageReady(!isHomePageLoading && !!account && !!organizationId, {
    canvas_count: canvases.length,
    folder_count: canvasFolders.length,
    failed: !!canvasError,
  });

  if (isHomePageLoading) {
    return <LoadingView />;
  }

  if (!account || !organizationId) {
    return <ErrorView />;
  }

  if (canvases.length === 0 && canvasFolders.length === 0 && !canvasError) {
    return <Navigate to={`/${organizationId}/apps/new`} replace />;
  }

  return (
    <HomePageShell>
      <div className="mx-auto w-full max-w-6xl p-8">
        <Header />

        <div className="mb-6">
          <CanvasToolbar searchQuery={searchQuery} setSearchQuery={setSearchQuery} />
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
            onEditCanvas={openEdit}
            canUpdateCanvases={canUpdateCanvases}
            canDeleteCanvases={canDeleteCanvases}
            permissionsLoading={permissionsLoading}
          />
        )}
      </div>

      <EditAppModal
        open={isEditAppModalOpen}
        initialName={editingCanvas?.name ?? ""}
        initialDescription={editingCanvas?.description}
        isSaving={isEditAppSaving}
        onClose={closeEdit}
        onSave={saveApp}
      />
    </HomePageShell>
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
  const folderedLayout = buildFolderedLayout(filteredCanvases, canvasFolders, Boolean(searchQuery));

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
        <section className={`${CANVAS_FOLDER_SECTION_SHELL_CLASS} bg-slate-950/5`}>
          <CanvasCardsGrid
            canvases={folderedLayout.unfiledCanvases}
            canvasFolders={canvasFolders}
            organizationId={organizationId}
            onEditCanvas={onEditCanvas}
            canUpdateCanvases={canUpdateCanvases}
            canDeleteCanvases={canDeleteCanvases}
            permissionsLoading={permissionsLoading}
          />
        </section>
      ) : null}
    </div>
  );
}

interface FolderedCanvasLayout {
  canvasesByFolderID: Map<string, CanvasCardData[]>;
  unfiledCanvases: CanvasCardData[];
  visibleFolders: CanvasFolderData[];
}

function buildFolderedLayout(
  filteredCanvases: CanvasCardData[],
  canvasFolders: CanvasFolderData[],
  hasSearchQuery: boolean,
): FolderedCanvasLayout {
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

  const visibleFolders = hasSearchQuery
    ? canvasFolders.filter((folder) => (canvasesByFolderID.get(folder.id) || []).length > 0)
    : canvasFolders;

  return { canvasesByFolderID, unfiledCanvases, visibleFolders };
}

function CanvasesSearchEmptyState() {
  return (
    <div className="text-center py-12">
      <Palette className="mx-auto text-gray-400 mb-4" size={48} aria-hidden />
      <Heading level={3} className="text-lg text-gray-800 dark:text-white mb-2">
        No apps found
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
        No apps yet
      </Heading>
    </div>
  );
}

function LoadingView() {
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600"></div>
      <p className="ml-3 text-gray-500">Loading...</p>
    </div>
  );
}

function ErrorView() {
  return (
    <div className="text-center py-8">
      <p className="text-gray-500">Unable to load user information</p>
    </div>
  );
}

function Header() {
  return (
    <div className="mb-6 flex items-center justify-between">
      <div>
        <Heading level={2} className="!text-2xl mb-1">
          Apps
        </Heading>
        <Text className="text-gray-800 dark:text-gray-400">
          Overview of all mapped automations across your organization.
        </Text>
      </div>
    </div>
  );
}
