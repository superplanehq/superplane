import type { CanvasFoldersCanvasFolder } from "@/api-client";
import { normalizeCanvasFolderColor, useCanvasFolders } from "@/hooks/useCanvasData";
import { useMemo } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import type { CanvasFolderData } from "./types";

export function useNewAppFolderContext() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const [searchParams] = useSearchParams();
  const folderId = searchParams.get("folderId") || undefined;
  const { data: canvasFolders = [] } = useCanvasFolders(organizationId || "");

  const folder = useMemo(() => {
    if (!folderId) {
      return undefined;
    }

    const matchingFolder = canvasFolders.find((item) => item.metadata?.id === folderId);
    return matchingFolder ? toCanvasFolderData(matchingFolder) : undefined;
  }, [canvasFolders, folderId]);

  return { folder };
}

function toCanvasFolderData(folder: CanvasFoldersCanvasFolder): CanvasFolderData | undefined {
  const id = folder.metadata?.id || "";
  const title = folder.spec?.title || "";
  if (!id || !title) {
    return undefined;
  }

  return {
    id,
    title,
    backgroundColor: normalizeCanvasFolderColor(folder.spec?.backgroundColor),
    canvasIds: folder.spec?.canvases?.map((canvas) => canvas.id || "").filter(Boolean) || [],
  };
}
