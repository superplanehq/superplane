import { useCallback, useEffect, useState, type FormEvent } from "react";

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
import { Textarea } from "@/components/ui/textarea";

import type { NewFactoryInput } from "./factoryTypes";

interface NewSoftwareFactoryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (input: NewFactoryInput) => void;
}

const emptyFactory: NewFactoryInput = { name: "", description: "" };

export function NewSoftwareFactoryDialog({ open, onOpenChange, onCreate }: NewSoftwareFactoryDialogProps) {
  const [draft, setDraft] = useState<NewFactoryInput>(emptyFactory);

  useEffect(() => {
    if (!open) setDraft(emptyFactory);
  }, [open]);

  const createFactory = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (draft.name.trim().length === 0) return;

      onCreate({
        name: draft.name.trim(),
        description: draft.description.trim(),
      });
      onOpenChange(false);
    },
    [draft, onCreate, onOpenChange],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={createFactory}>
          <DialogHeader>
            <DialogTitle>Create Software Factory</DialogTitle>
            <DialogDescription>
              Create the workspace first. You can add an Automation and connect repositories afterward.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-5 space-y-4">
            <div className="space-y-2">
              <Label htmlFor="factory-name">Name</Label>
              <Input
                id="factory-name"
                autoFocus
                value={draft.name}
                placeholder="Payments Factory"
                onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="factory-description">
                Description <span className="font-normal text-gray-500">Optional</span>
              </Label>
              <Textarea
                id="factory-description"
                rows={4}
                value={draft.description}
                placeholder="What kind of software work should this Factory handle?"
                onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))}
              />
            </div>
          </div>

          <DialogFooter className="mt-6">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={draft.name.trim().length === 0}>
              Create Factory
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
