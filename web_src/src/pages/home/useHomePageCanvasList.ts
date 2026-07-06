import type { CanvasFoldersCanvasFolder, CanvasesCanvasSummary } from "@/api-client";
import { normalizeCanvasFolderColor, useCanvasFolders, useCanvases } from "@/hooks/useCanvasData";
import type { CanvasCardData, CanvasFolderData } from "./types";

const compareByName = <T extends { name: string }>(left: T, right: T) => left.name.localeCompare(right.name);

function formatCanvasDate(value?: string) {
  if (!value) return "Unknown";
  return new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}

function toCanvasCardData(canvas: CanvasesCanvasSummary): CanvasCardData | null {
  const { id, name, createdBy } = canvas;
  const createdByName = createdBy?.name;
  if (!id || !name || !createdByName) {
    return null;
  }

  return {
    id,
    name,
    description: canvas.description,
    createdAt: formatCanvasDate(canvas.createdAt),
    canvasFolderId: canvas.folderId || undefined,
    createdBy: { name: createdByName },
    nodes: canvas.nodes || [],
    edges: canvas.edges || [],
  };
}

function toCanvasFolderData(folder: CanvasFoldersCanvasFolder): CanvasFolderData | null {
  const id = folder.metadata?.id || "";
  const title = folder.spec?.title || "";
  if (!id || !title) {
    return null;
  }

  return {
    id,
    title,
    backgroundColor: normalizeCanvasFolderColor(folder.spec?.backgroundColor),
    canvasIds: folder.spec?.canvases?.map((canvas) => canvas.id || "").filter(Boolean) || [],
  };
}

function filterCanvasesByQuery(canvases: CanvasCardData[], searchQuery: string) {
  const normalizedQuery = searchQuery.toLowerCase();
  return canvases.filter(
    (canvas) =>
      canvas.name.toLowerCase().includes(normalizedQuery) ||
      canvas.description?.toLowerCase().includes(normalizedQuery),
  );
}

export function useHomePageCanvasList(organizationId: string | undefined, searchQuery: string) {
  const {
    data: canvasesData = [],
    isLoading: canvasesLoading,
    isFetching: canvasesFetching,
    error: canvasesApiError,
  } = useCanvases(organizationId || "");
  const {
    data: canvasFoldersData = [],
    isLoading: canvasFoldersLoading,
    isFetching: canvasFoldersFetching,
    error: canvasFoldersApiError,
  } = useCanvasFolders(organizationId || "");

  const canvases = (canvasesData || [])
    .map(toCanvasCardData)
    .filter((canvas): canvas is CanvasCardData => canvas !== null)
    .sort(compareByName);

  const canvasFolders = (canvasFoldersData || [])
    .map(toCanvasFolderData)
    .filter((folder): folder is CanvasFolderData => folder !== null);

  return {
    canvases,
    canvasFolders,
    filteredCanvases: filterCanvasesByQuery(canvases, searchQuery),
    isLoading: canvasesLoading || canvasFoldersLoading,
    isFetching: canvasesFetching || canvasFoldersFetching,
    canvasError: canvasesApiError || canvasFoldersApiError ? "Failed to fetch canvases. Please try again later." : null,
  };
}
