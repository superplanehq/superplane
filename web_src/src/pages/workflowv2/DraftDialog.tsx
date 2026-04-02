import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface DraftDialogProps {
  isOpen: boolean;
  draftLastEditedAt?: string | null;
  onContinueDraft: () => void;
  onStartFresh: () => void;
  onClose: () => void;
}

function formatDraftDate(at: string | null | undefined): string {
  if (!at) return "Unknown time";
  const d = new Date(at);
  if (Number.isNaN(d.getTime())) return "Unknown time";
  return d.toLocaleString();
}

export function DraftDialog({ isOpen, draftLastEditedAt, onContinueDraft, onStartFresh, onClose }: DraftDialogProps) {
  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>You have a saved draft</DialogTitle>
          <DialogDescription>
            You have an existing draft last edited on {formatDraftDate(draftLastEditedAt)}. Would you like to continue
            working on it or start fresh from the current live version?
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="flex-col gap-2 sm:flex-row">
          <Button type="button" variant="outline" onClick={onStartFresh} className="w-full sm:w-auto">
            Start Fresh
          </Button>
          <Button type="button" onClick={onContinueDraft} className="w-full sm:w-auto">
            Continue Draft
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
