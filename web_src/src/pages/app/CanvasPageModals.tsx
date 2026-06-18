import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface CanvasPageModalsProps {
  canvasDeletedRemotely: boolean;
  onGoToCanvases: () => void;
}

export function CanvasPageModals({ canvasDeletedRemotely, onGoToCanvases }: CanvasPageModalsProps) {
  return (
    <Dialog open={canvasDeletedRemotely} onOpenChange={() => {}}>
      <DialogContent showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Canvas deleted</DialogTitle>
          <DialogDescription>
            This canvas was deleted from another session. You can no longer edit or run it.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button onClick={onGoToCanvases}>Go to canvases</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
