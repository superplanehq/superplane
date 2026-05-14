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
import { useCallback, useState } from "react";
import type { PanelDef } from "./panelRegistry";

function toSlug(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
}

interface AddPanelDialogProps {
  open: boolean;
  panelDef: PanelDef | null;
  onConfirm: (def: PanelDef, name: string) => void;
  onCancel: () => void;
}

export function AddPanelDialog({ open, panelDef, onConfirm, onCancel }: AddPanelDialogProps) {
  const [name, setName] = useState("");
  const slug = toSlug(name);
  const isValid = slug.length > 0;

  const handleConfirm = useCallback(() => {
    if (panelDef && isValid) {
      onConfirm(panelDef, name.trim());
      setName("");
    }
  }, [panelDef, isValid, name, onConfirm]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && isValid) {
        handleConfirm();
      }
    },
    [handleConfirm, isValid],
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          setName("");
          onCancel();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add panel</DialogTitle>
          <DialogDescription>Give your panel a name. This will be used as its identifier in the git repository.</DialogDescription>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="panel-name">Name</Label>
            <Input
              id="panel-name"
              placeholder="e.g. Pipeline Status"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={handleKeyDown}
              autoFocus
              data-testid="add-panel-name-input"
            />
          </div>
          {name.trim() && (
            <p className="text-xs text-slate-500">
              ID: <code className="rounded bg-slate-100 px-1 py-0.5">{slug || "—"}</code>
            </p>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button onClick={handleConfirm} disabled={!isValid} data-testid="add-panel-confirm">
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
