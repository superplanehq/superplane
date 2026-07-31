import { Loader2 } from "lucide-react";

export function CanvasPageLoadingOverlay({ message, testId }: { message: string; testId?: string }) {
  return (
    <div
      className="absolute inset-0 z-20 flex items-center justify-center bg-surface-raised/70 backdrop-blur-[1px]"
      data-testid={testId}
    >
      <div className="flex items-center gap-2 rounded-md border border-edge-subtle bg-surface-raised px-3 py-2 text-sm text-content-secondary shadow-sm">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>{message}</span>
      </div>
    </div>
  );
}
