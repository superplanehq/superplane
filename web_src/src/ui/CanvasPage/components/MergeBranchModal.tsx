import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CANVAS_MAIN_BRANCH } from "@/lib/canvas-branches";

export type MergeBranchModalSubmit = {
  commitMessage: string;
};

export function defaultMergeCommitMessage(branchName: string): string {
  const trimmed = branchName.trim() || CANVAS_MAIN_BRANCH;
  return `Merge branch '${trimmed}'`;
}

type MergeBranchModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sourceBranchName: string;
  pending?: boolean;
  onSubmit: (input: MergeBranchModalSubmit) => void | Promise<void>;
};

export function MergeBranchModal({
  open,
  onOpenChange,
  sourceBranchName,
  pending = false,
  onSubmit,
}: MergeBranchModalProps) {
  const [commitMessage, setCommitMessage] = useState("");

  useEffect(() => {
    if (open) {
      setCommitMessage(defaultMergeCommitMessage(sourceBranchName));
    }
  }, [open, sourceBranchName]);

  const canSubmit = commitMessage.trim().length > 0 && !pending;

  const handleSubmit = async () => {
    if (!canSubmit) {
      return;
    }

    await onSubmit({ commitMessage: commitMessage.trim() });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" data-testid="merge-branch-modal">
        <DialogHeader>
          <DialogTitle>Merge branch</DialogTitle>
          <DialogDescription>
            Merge <span className="font-medium text-slate-900">{sourceBranchName}</span> into{" "}
            <span className="font-medium text-slate-900">{CANVAS_MAIN_BRANCH}</span>.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 py-1">
          <Label htmlFor="merge-commit-message">Commit message</Label>
          <Input
            id="merge-commit-message"
            value={commitMessage}
            onChange={(event) => setCommitMessage(event.target.value)}
            placeholder="Describe the merge"
            autoFocus
            data-testid="merge-commit-message-input"
          />
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={() => void handleSubmit()}
            disabled={!canSubmit}
            data-testid="confirm-merge-button"
          >
            Merge
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
