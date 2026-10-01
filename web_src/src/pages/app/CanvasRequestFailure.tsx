import type { ReactElement } from "react";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

import { isCanvasLoadNotFoundError } from "./workflowPageHelpers";

type EarlyCanvasViewProps = {
  canvas: unknown;
  canvasLoading: boolean;
  canvasError: unknown;
  bootstrapLoading: boolean;
  showDraftCanvasLoadingOverlay: boolean;
  onRetry: () => void;
};

export function EarlyCanvasView(props: EarlyCanvasViewProps): ReactElement | null {
  if (isCanvasRequestFailure(props)) {
    return <CanvasRequestFailure onRetry={props.onRetry} />;
  }

  if (props.bootstrapLoading) {
    return <CanvasBootstrapLoading />;
  }

  if (!props.canvas && !props.canvasLoading && !props.showDraftCanvasLoadingOverlay) {
    return <CanvasNotFound />;
  }

  return null;
}

export function CanvasRequestFailure({ onRetry }: { onRetry: () => void }): ReactElement {
  return (
    <div
      role="alert"
      data-testid="canvas-request-failure"
      className={cn("flex h-screen items-center justify-center bg-gray-50 px-6", appDarkModeClasses.surface)}
    >
      <div className="flex max-w-md flex-col items-center gap-3 text-center">
        <h1 className={cn("text-lg font-semibold text-gray-800", appDarkModeClasses.textPrimary)}>
          Could not load this canvas
        </h1>
        <p className={cn("text-sm text-gray-500", appDarkModeClasses.textSecondary)}>The request failed.</p>
        <Button type="button" onClick={onRetry}>
          Try again
        </Button>
      </div>
    </div>
  );
}

function isCanvasRequestFailure({
  canvas,
  canvasLoading,
  canvasError,
  showDraftCanvasLoadingOverlay,
}: EarlyCanvasViewProps): boolean {
  return (
    !canvas &&
    !canvasLoading &&
    !showDraftCanvasLoadingOverlay &&
    canvasError != null &&
    !isCanvasLoadNotFoundError(canvasError)
  );
}

function CanvasBootstrapLoading(): ReactElement {
  return (
    <div className="flex items-center justify-center h-screen">
      <div className="flex flex-col items-center gap-3">
        <Loader2 className="h-8 w-8 animate-spin text-gray-500" />
        <p className="text-sm text-gray-500">Loading canvas...</p>
      </div>
    </div>
  );
}

function CanvasNotFound(): ReactElement {
  return (
    <div className="flex items-center justify-center h-screen">
      <div className="flex flex-col items-center gap-4">
        <h1 className="text-4xl font-bold text-gray-700">404</h1>
        <p className="text-sm text-gray-500">Canvas not found</p>
        <p className="text-sm text-gray-400">
          This canvas may have been deleted or you may not have permission to view it.
        </p>
      </div>
    </div>
  );
}
