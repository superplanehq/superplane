import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useCanvasId } from "@/hooks/useCanvasId";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { ScrollText } from "lucide-react";
import { useCallback, useState } from "react";
import { LiveLogStreamView } from "./LiveLogStreamView";
import type { RunnerLiveLogDialogProps } from "./types";

export function RunnerLiveLogDialog({ canvasMode, executionId }: RunnerLiveLogDialogProps) {
  const organizationId = useOrganizationId();
  const canvasId = useCanvasId();
  const [open, setOpen] = useState(false);

  const canShow = canvasMode === "live" && !!organizationId && !!canvasId && !!executionId;
  if (!canShow) {
    return null;
  }

  const handleOpen = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setOpen(true);
  }, []);

  return (
    <>
      <div className="flex items-center justify-center gap-1 cursor-pointer py-1.5" onClick={handleOpen}>
        <ScrollText className="h-4 w-4" /> View logs
      </div>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          size="large"
          className="flex max-h-[min(90vh,720px)] w-[min(90vw,56rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-none"
          onClick={(e) => e.stopPropagation()}
        >
          <DialogHeader className="shrink-0 border-b border-gray-200 px-4 py-3 text-left">
            <DialogTitle>Logs</DialogTitle>
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-hidden">
            {open ? <LiveLogStreamView executionId={executionId} /> : null}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
