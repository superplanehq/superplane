import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { PermissionTooltip } from "@/components/PermissionGate";
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
import { useCanvasGroups, useCanvases } from "../../hooks/useCanvasData";
import { Button } from "@/components/ui/button";
import { useCreateCanvasModalState } from "./useCreateCanvasModalState";
import type { CanvasesCanvas, CanvasesCanvasGroup } from "@/api-client";
import { CanvasCard } from "./CanvasCard";
import { CanvasGroupSection } from "./CanvasGroupSection";
import { asCanvasGroupColor, compareByName, type CanvasCardData, type CanvasGroupData } from "./shared";

export const HomePage = () => {
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
      id: canvas.metadata?.id ?? "",
      name: canvas.metadata?.name ?? "",
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
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          {groupedLayout.ungroupedCanvases.map((canvas) => (
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
