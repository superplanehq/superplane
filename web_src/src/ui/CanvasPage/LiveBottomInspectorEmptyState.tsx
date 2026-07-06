import { Button } from "@/components/ui/button";
import { MousePointer, X } from "lucide-react";

interface LiveBottomInspectorEmptyStateProps {
  onClose: () => void;
}

export function LiveBottomInspectorEmptyState({ onClose }: LiveBottomInspectorEmptyStateProps) {
  return (
    <div
      className="flex h-full min-h-0 flex-1 flex-col overflow-hidden bg-white dark:bg-gray-900"
      data-testid="live-bottom-inspector-empty"
    >
      <div className="flex h-9 shrink-0 items-stretch justify-end border-b border-slate-200 dark:border-gray-800/70">
        <div className="flex items-center px-1">
          <Button type="button" variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onClose}>
            <X className="size-3.5" />
          </Button>
        </div>
      </div>
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-4 text-center">
        <MousePointer className="h-5 w-5 text-gray-400 dark:text-gray-500" strokeWidth={1.5} aria-hidden />
        <p className="text-[13px] text-gray-500 dark:text-gray-400">Select component to inspect</p>
      </div>
    </div>
  );
}
