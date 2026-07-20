import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

export type CanvasSaveResult = {
  status: "saved" | "replaced" | "stale";
  workflow: CanvasesCanvas;
  savingVersionId?: string;
  matchesCurrentCanvas: boolean;
  hasQueuedFollowUp: boolean;
  response?: {
    data?: {
      version?: CanvasesCanvasVersion;
    };
  };
};

export type QueuedCanvasSaveRequest = {
  workflow: CanvasesCanvas;
  savingVersionId?: string;
  resolve: (result: CanvasSaveResult) => void;
  reject: (error: unknown) => void;
};

export type CanvasEchoRelease = () => void;
