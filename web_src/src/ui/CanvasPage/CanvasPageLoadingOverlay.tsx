import { Loader2 } from "lucide-react";

export function CanvasPageLoadingOverlay({ message, testId }: { message: string; testId?: string }) {
  return (
    <div
      className="absolute inset-0 z-20 flex items-center justify-center bg-white/70 backdrop-blur-[1px] dark:bg-gray-900/70"
      data-testid={testId}
    >
      <div className="flex items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-600 shadow-sm dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>{message}</span>
      </div>
    </div>
  );
}
