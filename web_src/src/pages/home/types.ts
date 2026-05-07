import type { SuperplaneComponentsEdge, SuperplaneComponentsNode } from "@/api-client";
import type { CanvasFolderColor } from "@/hooks/useCanvasData";

export interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  canvasFolderId?: string;
  createdBy?: { id?: string; name?: string };
  nodes?: SuperplaneComponentsNode[];
  edges?: SuperplaneComponentsEdge[];
}

export interface CanvasFolderData {
  id: string;
  title: string;
  backgroundColor: CanvasFolderColor;
  canvasIds: string[];
}
