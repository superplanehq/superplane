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
import { useState } from "react";

import { lineStageNameError, type LineBoardStage } from "../lib/lineBoardStages";

/** Names a draft stage. The stage is created when the stage edits are saved. */
export function AddLineStageDialog({
  stages,
  open,
  onClose,
  onAdd,
}: {
  stages: LineBoardStage[];
  open: boolean;
  onClose: () => void;
  onAdd: (name: string) => void;
}) {
  const [name, setName] = useState("");
  const trimmedName = name.trim();
  const nameError = lineStageNameError(name, stages);

  const close = () => {
    setName("");
    onClose();
  };

  const add = () => {
    if (nameError) {
      return;
    }
    setName("");
    onAdd(trimmedName);
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && close()}>
      <DialogContent className="sm:max-w-md" data-testid="add-line-stage-dialog">
        <DialogHeader>
          <DialogTitle>Add stage</DialogTitle>
          <DialogDescription>The stage goes before Done. It is created when you save your changes.</DialogDescription>
        </DialogHeader>
        <form
          className="space-y-2"
          onSubmit={(event) => {
            event.preventDefault();
            add();
          }}
        >
          <Label htmlFor="line-stage-name">Name</Label>
          <Input
            id="line-stage-name"
            value={name}
            maxLength={64}
            autoFocus
            onChange={(event) => setName(event.target.value)}
          />
          {trimmedName && nameError ? <p className="text-sm text-destructive">{nameError}</p> : null}
        </form>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="button" onClick={add} disabled={Boolean(nameError)}>
            Add stage
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
