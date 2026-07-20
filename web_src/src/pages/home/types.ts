import type { ComponentsEdge, SuperplaneComponentsNode } from "@/api-client";
import type { CanvasFolderColor } from "@/hooks/useCanvasData";

export interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  canvasFolderId?: string;
  isStarred?: boolean;
  starredAt?: string;
  createdBy: { name: string };
  nodes?: SuperplaneComponentsNode[];
  edges?: ComponentsEdge[];
}

export interface CanvasFolderData {
  id: string;
  title: string;
  backgroundColor: CanvasFolderColor;
  canvasIds: string[];
}
