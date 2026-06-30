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

export type CommitBranchModalSubmit = {
  commitMessage: string;
  newBranchName?: string;
};

type CommitBranchModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentBranchName: string;
  pending?: boolean;
  onSubmit: (input: CommitBranchModalSubmit) => void | Promise<void>;
};

export function CommitBranchModal({
  open,
  onOpenChange,
  currentBranchName,
  pending = false,
  onSubmit,
}: CommitBranchModalProps) {
  const [commitMessage, setCommitMessage] = useState("");
  const [target, setTarget] = useState<"current" | "new">("current");
  const [newBranchName, setNewBranchName] = useState("");

  useEffect(() => {
    if (!open) {
      setCommitMessage("");
      setTarget("current");
      setNewBranchName("");
    }
  }, [open]);

  const canSubmit =
    commitMessage.trim().length > 0 && !pending && (target === "current" || newBranchName.trim().length > 0);

  const handleSubmit = async () => {
    if (!canSubmit) {
      return;
    }

    await onSubmit({
      commitMessage: commitMessage.trim(),
      newBranchName: target === "new" ? newBranchName.trim() : undefined,
    });
  };

  const branchLabel = currentBranchName || CANVAS_MAIN_BRANCH;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md" data-testid="commit-branch-modal">
        <DialogHeader>
          <DialogTitle>Commit changes</DialogTitle>
          <DialogDescription>
            Create a commit on <span className="font-medium text-slate-900">{branchLabel}</span> or start a new branch.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-1">
          <div className="space-y-2">
            <Label htmlFor="commit-message">Commit message</Label>
            <Input
              id="commit-message"
              value={commitMessage}
              onChange={(event) => setCommitMessage(event.target.value)}
              placeholder="Describe your changes"
              autoFocus
              data-testid="commit-message-input"
            />
          </div>

          <div className="space-y-2">
            <Label>Destination</Label>
            <div className="space-y-2 text-sm">
              <label className="flex cursor-pointer items-center gap-2">
                <input
                  type="radio"
                  name="commit-target"
                  checked={target === "current"}
                  onChange={() => setTarget("current")}
                />
                <span>
                  Commit to <span className="font-medium">{branchLabel}</span>
                </span>
              </label>
              <label className="flex cursor-pointer items-center gap-2">
                <input type="radio" name="commit-target" checked={target === "new"} onChange={() => setTarget("new")} />
                <span>Create a new branch</span>
              </label>
            </div>
          </div>

          {target === "new" ? (
            <div className="space-y-2">
              <Label htmlFor="new-branch-name">New branch name</Label>
              <Input
                id="new-branch-name"
                value={newBranchName}
                onChange={(event) => setNewBranchName(event.target.value)}
                placeholder="feature/my-change"
                data-testid="new-branch-name-input"
              />
            </div>
          ) : null}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={() => void handleSubmit()}
            disabled={!canSubmit}
            data-testid="confirm-commit-button"
          >
            Commit
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
