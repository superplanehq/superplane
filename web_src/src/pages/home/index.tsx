import { usePermissions } from "@/contexts/usePermissions";
import { useUpdateCanvasPreference } from "@/hooks/useCanvasData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { Palette } from "lucide-react";
import { useState } from "react";
import { Navigate, useParams } from "react-router";
import { Heading } from "../../components/Heading/heading";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/useAccount";
import { CanvasCardsGrid } from "./CanvasCardsGrid";
import { CanvasToolbar } from "./CanvasToolbar";
import { EditAppModal } from "./EditAppModal";
import { HomePageShell } from "./HomePageShell";
import { RequireClassicAppsSurface } from "./RequireClassicAppsSurface";
import { applyCanvasAppPreferences } from "./canvasAppPreferencePresentation";
import type { CanvasCardData } from "./types";
import { useEditApp } from "./useEditApp";
import { useHomePageCanvasList } from "./useHomePageCanvasList";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

export function HomePage() {
  return (
    <RequireClassicAppsSurface>
      <ClassicHomePage />
    </RequireClassicAppsSurface>
  );
}

function ClassicHomePage() {
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

  const { canvases, filteredCanvases, isLoading, isFetching, canvasError } = useHomePageCanvasList(
    organizationId,
    searchQuery,
  );
  const updateCanvasPreference = useUpdateCanvasPreference(organizationId || "");
  const preferredFilteredCanvases = applyCanvasAppPreferences(filteredCanvases);
  const canCreateCanvases = canAct("canvases", "create");
  const canUpdateCanvases = canAct("canvases", "update");
  const canDeleteCanvases = canAct("canvases", "delete");

  const isHomePageLoading = isLoading || (isFetching && canvases.length === 0);
  useReportPageReady(!isHomePageLoading && !!account && !!organizationId, {
    canvas_count: canvases.length,
    failed: !!canvasError,
  });

  if (isHomePageLoading) {
    return <LoadingView />;
  }

  if (!account || !organizationId) {
    return <ErrorView />;
  }

  if (canvases.length === 0 && !canvasError && canCreateCanvases) {
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
          <div className="rounded border border-red-300 bg-white px-4 py-2 text-red-500 dark:border-red-800 dark:bg-gray-800 dark:text-red-400">
            <Text>{canvasError}</Text>
          </div>
        ) : (
          <Content
            filteredCanvases={filteredCanvases}
            preferredFilteredCanvases={preferredFilteredCanvases}
            organizationId={organizationId}
            searchQuery={searchQuery}
            onEditCanvas={openEdit}
            onToggleStar={(canvasId, starred) => updateCanvasPreference.mutate({ canvasId, starred })}
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
  preferredFilteredCanvases,
  organizationId,
  searchQuery,
  onEditCanvas,
  onToggleStar,
  canUpdateCanvases,
  canDeleteCanvases,
  permissionsLoading,
}: {
  filteredCanvases: CanvasCardData[];
  preferredFilteredCanvases: CanvasCardData[];
  organizationId: string;
  searchQuery: string;
  onEditCanvas: (canvas: CanvasCardData) => void;
  onToggleStar: (canvasId: string, starred: boolean) => void;
  canUpdateCanvases: boolean;
  canDeleteCanvases: boolean;
  permissionsLoading: boolean;
}) {
  if (filteredCanvases.length === 0) {
    return searchQuery ? <CanvasesSearchEmptyState /> : <CanvasesEmptyState />;
  }

  return (
    <CanvasCardsGrid
      canvases={preferredFilteredCanvases}
      organizationId={organizationId}
      onEditCanvas={onEditCanvas}
      onToggleStar={onToggleStar}
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
    <div className={cn("min-h-screen flex items-center justify-center bg-slate-100", appDarkModeClasses.surface)}>
      <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600 dark:border-blue-400"></div>
      <p className="ml-3 text-gray-500 dark:text-gray-400">Loading...</p>
    </div>
  );
}

function ErrorView() {
  return (
    <div className={cn("py-8 text-center bg-slate-100 min-h-screen", appDarkModeClasses.surface)}>
      <p className="text-gray-500 dark:text-gray-400">Unable to load user information</p>
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
