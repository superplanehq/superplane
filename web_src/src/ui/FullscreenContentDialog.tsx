import type { ReactNode } from "react";

import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

const FULLSCREEN_DIALOG_SIZE_CLASSES = {
  /** Run inspector / code blocks */
  default: "h-[80vh] w-[60vw] max-w-[60vw]",
  /** Wide diagrams (Mermaid) */
  wide: "h-[90vh] w-[90vw] max-w-[90vw]",
} as const;

export type FullscreenContentDialogSize = keyof typeof FULLSCREEN_DIALOG_SIZE_CLASSES;

/**
 * Fullscreen content shell shared by run-inspector OUTPUT expand,
 * markdown code-block expand, and Mermaid diagram expand.
 */
export function FullscreenContentDialog({
  open,
  onOpenChange,
  title,
  headerActions,
  bodyClassName,
  size = "default",
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  headerActions?: ReactNode;
  bodyClassName?: string;
  size?: FullscreenContentDialogSize;
  children: ReactNode;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className={cn("flex flex-col gap-0 overflow-hidden p-0", FULLSCREEN_DIALOG_SIZE_CLASSES[size])}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 pr-10 dark:border-gray-800 dark:bg-gray-900">
          <DialogTitle className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
            {title}
          </DialogTitle>
          {headerActions ? <span className="flex items-center gap-0.5">{headerActions}</span> : null}
        </div>
        <div className={cn("min-h-0 flex-1 overflow-auto p-3", bodyClassName)}>{children}</div>
      </DialogContent>
    </Dialog>
  );
}
