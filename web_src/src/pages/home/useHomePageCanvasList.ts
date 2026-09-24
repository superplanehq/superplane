import type { CanvasesCanvasSummary } from "@/api-client";
import { useCanvases } from "@/hooks/useCanvasData";
import type { CanvasCardData } from "./types";

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
    isStarred: canvas.starred ?? false,
    starredAt: canvas.starredAt,
    createdBy: { name: createdByName },
    nodes: canvas.nodes || [],
    edges: canvas.edges || [],
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

  const canvases = (canvasesData || [])
    .map(toCanvasCardData)
    .filter((canvas): canvas is CanvasCardData => canvas !== null)
    .sort(compareByName);

  return {
    canvases,
    filteredCanvases: filterCanvasesByQuery(canvases, searchQuery),
    isLoading: canvasesLoading,
    isFetching: canvasesFetching,
    canvasError: canvasesApiError ? "Failed to fetch canvases. Please try again later." : null,
  };
}
