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
import { useEffect, useState } from "react";

export type BacklogSettings = {
  name: string;
  size: number | null;
};

interface BacklogSettingsDialogProps {
  open: boolean;
  name: string;
  size: number | null;
  onSave: (settings: BacklogSettings) => void;
  onClose: () => void;
}

const SIZE_ERROR = "Enter a whole number of 1 or more.";

function parseBacklogSize(value: string): { size: number | null; error?: string } {
  const trimmed = value.trim();
  if (!trimmed) {
    return { size: null };
  }
  if (!/^\d+$/.test(trimmed)) {
    return { size: null, error: SIZE_ERROR };
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (parsed < 1) {
    return { size: null, error: SIZE_ERROR };
  }
  return { size: parsed };
}

export function BacklogSettingsDialog({ open, name, size, onSave, onClose }: BacklogSettingsDialogProps) {
  const [draftName, setDraftName] = useState(name);
  const [draftSize, setDraftSize] = useState(size == null ? "" : String(size));
  const [nameError, setNameError] = useState("");
  const [sizeError, setSizeError] = useState("");

  useEffect(() => {
    if (!open) {
      return;
    }
    setDraftName(name);
    setDraftSize(size == null ? "" : String(size));
    setNameError("");
    setSizeError("");
  }, [open, name, size]);

  const handleSave = () => {
    const trimmedName = draftName.trim();
    if (!trimmedName) {
      setNameError("Enter a name.");
      return;
    }
    const parsed = parseBacklogSize(draftSize);
    if (parsed.error) {
      setSizeError(parsed.error);
      return;
    }
    onSave({ name: trimmedName, size: parsed.size });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-md" data-testid="lines-backlog-settings">
        <DialogHeader>
          <DialogTitle>Edit backlog</DialogTitle>
          <DialogDescription>Set the column name and the maximum number of tasks.</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="lines-backlog-settings-name">Name</Label>
            <Input
              id="lines-backlog-settings-name"
              value={draftName}
              onChange={(event) => {
                setDraftName(event.target.value);
                if (nameError) {
                  setNameError("");
                }
              }}
              autoFocus
            />
            {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="lines-backlog-settings-size">Size</Label>
            <Input
              id="lines-backlog-settings-size"
              inputMode="numeric"
              value={draftSize}
              onChange={(event) => {
                setDraftSize(event.target.value);
                if (sizeError) {
                  setSizeError("");
                }
              }}
            />
            <p className="text-xs text-muted-foreground">Maximum number of tasks. Leave empty for no limit.</p>
            {sizeError ? <p className="text-xs text-red-600">{sizeError}</p> : null}
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} data-testid="lines-backlog-settings-save">
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
