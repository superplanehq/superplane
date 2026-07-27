import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useState, type FormEvent } from "react";

import type { CreateWorkRequest } from "./types";

interface NewWorkDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreateWork?: (request: CreateWorkRequest) => void;
}

export function NewWorkDialog({ open, onOpenChange, onCreateWork }: NewWorkDialogProps) {
  const [title, setTitle] = useState("");
  const [goal, setGoal] = useState("");

  const closeDialog = () => {
    setTitle("");
    setGoal("");
    onOpenChange(false);
  };

  const submitWork = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedTitle = title.trim();
    if (!trimmedTitle) return;

    onCreateWork?.({ title: trimmedTitle, goal: goal.trim() });
    closeDialog();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (isOpen) {
          onOpenChange(true);
          return;
        }
        closeDialog();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <form onSubmit={submitWork} className="space-y-5">
          <DialogHeader>
            <DialogTitle>Start new work</DialogTitle>
            <DialogDescription>
              Give the factory an outcome. It will plan the work, assign agents, verify the change, and prepare a pull
              request.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="workspace-work-title">Work item</Label>
            <Input
              id="workspace-work-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="e.g. Add SSO session recovery"
              autoFocus
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="workspace-work-goal">Goal</Label>
            <Textarea
              id="workspace-work-goal"
              value={goal}
              onChange={(event) => setGoal(event.target.value)}
              placeholder="Describe the expected outcome and any constraints."
              className="min-h-24 resize-none"
            />
          </div>

          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline">
                Cancel
              </Button>
            </DialogClose>
            <Button type="submit" disabled={!title.trim()}>
              Start factory run
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
