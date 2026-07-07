import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useCallback, useState } from "react";
import { Icon } from "../../../components/Icon";
import { LiveLogStreamView } from "./LiveLogStreamView";
import type { RunnerLiveLogDialogProps } from "./types";

export function RunnerLiveLogDialog({ title, canvasMode, execution }: RunnerLiveLogDialogProps) {
  const [open, setOpen] = useState(false);

  const handleOpen = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setOpen(true);
  }, []);

  if (!execution) {
    return null;
  }

  if (canvasMode !== "live") {
    return null;
  }

  return (
    <>
      <div className="flex items-center justify-center gap-1 cursor-pointer py-1.5" onClick={handleOpen}>
        <Icon name="scroll-text" className="h-4 w-4" /> View logs
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          size="90vw"
          className="flex flex-col gap-0 overflow-hidden p-0"
          onClick={(e) => e.stopPropagation()}
        >
          <DialogHeader className="shrink-0 border-b border-gray-200 px-4 py-3 text-left dark:border-gray-800">
            <DialogTitle className="text-sm font-medium">{title}</DialogTitle>
          </DialogHeader>

          <div className="min-h-0 flex-1 bg-slate-50 dark:bg-gray-900">
            <LiveLogStreamView execution={execution} />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
