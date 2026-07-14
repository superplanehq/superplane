import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";

export function RunInspectorErrorSummaryCard({
  nodeName,
  message,
  onJump,
}: {
  nodeName: string;
  message: string;
  onJump: () => void;
}) {
  return (
    <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2.5 text-red-700 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-300">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-600 dark:text-red-300" />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-semibold text-red-800 dark:text-red-200">Errored at &quot;{nodeName}&quot;</p>
        <p className="mt-0.5 line-clamp-3 break-words text-xs text-red-700 dark:text-red-300">{message}</p>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="shrink-0 rounded-sm border-red-300 bg-white text-red-700 hover:bg-red-100 dark:border-red-800 dark:bg-gray-950 dark:text-red-300 dark:hover:bg-red-950"
        onClick={onJump}
      >
        Jump to error
      </Button>
    </div>
  );
}
