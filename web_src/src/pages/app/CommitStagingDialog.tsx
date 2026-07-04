import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useEffect, useState } from "react";

type CommitStagingDialogProps = {
  open: boolean;
  pending?: boolean;
  onOpenChange: (open: boolean) => void;
  onCommit: (commitMessage: string) => void | Promise<void>;
};

export function CommitStagingDialog({ open, pending, onOpenChange, onCommit }: CommitStagingDialogProps) {
  const [commitMessage, setCommitMessage] = useState("");

  useEffect(() => {
    if (!open) {
      setCommitMessage("");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Commit changes</DialogTitle>
          <DialogDescription>Describe what changed in this commit.</DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="commit-message">Commit message</Label>
          <Textarea
            id="commit-message"
            value={commitMessage}
            onChange={(event) => setCommitMessage(event.target.value)}
            placeholder="Update workflow triggers"
            rows={4}
            disabled={pending}
            data-testid="canvas-commit-message-input"
          />
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={() => void onCommit(commitMessage)}
            disabled={pending || !commitMessage.trim()}
            data-testid="canvas-commit-message-submit"
          >
            Commit
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
