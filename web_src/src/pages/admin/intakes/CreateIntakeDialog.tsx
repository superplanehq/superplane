import { useState } from "react";

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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useCreateAdminIntake } from "@/hooks/useAdminIntakeCatalog";
import { INTAKE_CATEGORIES } from "@/lib/intakeCatalog";

import {
  CREATE_INTAKE_HELP,
  INTAKE_CATEGORY_LABELS,
  intakeKeyFromName,
  type AdminIntakeEntry,
} from "./intakeCatalogModel";

interface CreateIntakeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (entry: AdminIntakeEntry) => void;
}

interface CreateForm {
  name: string;
  key: string;
  keyEdited: boolean;
  category: string;
  note: string;
}

const EMPTY_FORM: CreateForm = { name: "", key: "", keyEdited: false, category: "issue_tracking", note: "" };

export function CreateIntakeDialog({ open, onOpenChange, onCreated }: CreateIntakeDialogProps) {
  const [form, setForm] = useState<CreateForm>(EMPTY_FORM);
  const createIntake = useCreateAdminIntake();

  const close = (next: boolean) => {
    if (!next) {
      setForm(EMPTY_FORM);
      createIntake.reset();
    }
    onOpenChange(next);
  };

  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    createIntake.mutate(
      { name: form.name.trim(), key: form.key.trim(), category: form.category, status_note: form.note.trim() },
      {
        onSuccess: (entry) => {
          close(false);
          onCreated(entry);
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={submit} className="flex flex-col gap-4">
          <DialogHeader>
            <DialogTitle>Add intake</DialogTitle>
            <DialogDescription>Add an intake to the catalog so that admins can track its status.</DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="intake-name">Name</Label>
            <Input
              id="intake-name"
              value={form.name}
              placeholder="Linear issues"
              onChange={(event: React.ChangeEvent<HTMLInputElement>) => {
                const name = event.target.value;
                setForm((prev) => ({ ...prev, name, key: prev.keyEdited ? prev.key : intakeKeyFromName(name) }));
              }}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="intake-key">Key</Label>
            <Input
              id="intake-key"
              value={form.key}
              placeholder="linear-issues"
              className="font-mono"
              onChange={(event: React.ChangeEvent<HTMLInputElement>) =>
                setForm((prev) => ({ ...prev, key: event.target.value, keyEdited: true }))
              }
            />
            <p className="text-xs text-muted-foreground">Use lowercase letters, numbers, and dashes.</p>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="intake-category">Category</Label>
            <Select value={form.category} onValueChange={(category) => setForm((prev) => ({ ...prev, category }))}>
              <SelectTrigger id="intake-category" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {INTAKE_CATEGORIES.map((category) => (
                  <SelectItem key={category} value={category}>
                    {INTAKE_CATEGORY_LABELS[category]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="intake-note">Status note (optional)</Label>
            <Textarea
              id="intake-note"
              value={form.note}
              rows={3}
              onChange={(event: React.ChangeEvent<HTMLTextAreaElement>) =>
                setForm((prev) => ({ ...prev, note: event.target.value }))
              }
            />
          </div>

          <p className="text-xs text-muted-foreground">{CREATE_INTAKE_HELP}</p>
          {createIntake.error ? (
            <p role="alert" className="text-sm text-red-600 dark:text-red-400">
              {createIntake.error.message}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => close(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!form.name.trim() || !form.key.trim() || createIntake.isPending}>
              {createIntake.isPending ? "Adding..." : "Add intake"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
