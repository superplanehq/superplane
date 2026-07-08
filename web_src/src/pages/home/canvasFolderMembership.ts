import type { CanvasFolderData } from "./types";

export function appendCanvasToFolderMembership(folder: CanvasFolderData, canvasId: string) {
  return {
    folderId: folder.id,
    title: folder.title,
    backgroundColor: folder.backgroundColor,
    canvasIds: folder.canvasIds.includes(canvasId) ? folder.canvasIds : [...folder.canvasIds, canvasId],
  };
}
