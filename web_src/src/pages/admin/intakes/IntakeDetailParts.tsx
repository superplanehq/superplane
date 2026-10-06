import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

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
import { formatRelative } from "@/lib/datetime";
import { INTAKE_CATEGORIES } from "@/lib/intakeCatalog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { formatDate } from "../formatDate";
import { INTAKE_CATEGORY_LABELS, STATUS_NOTE_HELP, type AdminIntakeEntry } from "./intakeCatalogModel";

export function DetailCard({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-900">
      <header className="border-b border-slate-100 px-5 py-3 dark:border-gray-800">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">{title}</h2>
        {description ? <p className="text-xs text-slate-500 dark:text-gray-400">{description}</p> : null}
      </header>
      <div className="px-5 py-4">{children}</div>
    </section>
  );
}

interface StatusNoteEditorProps {
  entry: AdminIntakeEntry;
  saving: boolean;
  savedAt: number | null;
  onSave: (note: string) => void;
}

export function StatusNoteEditor({ entry, saving, savedAt, onSave }: StatusNoteEditorProps) {
  const [note, setNote] = useState(entry.status_note);
  const [dirty, setDirty] = useState(false);
  const lastEntryKey = useRef(entry.key);

  useEffect(() => {
    if (lastEntryKey.current === entry.key) return;
    lastEntryKey.current = entry.key;
    setNote(entry.status_note);
    setDirty(false);
  }, [entry.key, entry.status_note]);

  useEffect(() => {
    if (!dirty) {
      setNote(entry.status_note);
      return;
    }

    if (note.trim() === entry.status_note) {
      setDirty(false);
      setNote(entry.status_note);
    }
  }, [dirty, entry.status_note, note]);

  const save = () => {
    if (note.trim() !== entry.status_note) {
      onSave(note);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <Textarea
        aria-label="Status note"
        value={note}
        rows={4}
        placeholder="For example: Works for Jira Cloud. Jira Server is not tested."
        onChange={(event: React.ChangeEvent<HTMLTextAreaElement>) => {
          const next = event.target.value;
          setNote(next);
          setDirty(next.trim() !== entry.status_note);
        }}
        onBlur={save}
      />
      <div className="flex items-center justify-between text-xs text-slate-500 dark:text-gray-400">
        <span>{STATUS_NOTE_HELP}</span>
        <span aria-live="polite">{saving ? "Saving..." : savedAt ? `Saved ${formatRelative(savedAt)}` : ""}</span>
      </div>
    </div>
  );
}

interface MetadataRailProps {
  entry: AdminIntakeEntry;
  disabled: boolean;
  onChangeCategory: (category: string) => void;
}

export function MetadataRail({ entry, disabled, onChangeCategory }: MetadataRailProps) {
  const lastChange = entry.updated_by_name
    ? `${entry.updated_by_name}, ${formatRelative(entry.updated_at)}`
    : formatRelative(entry.updated_at);

  return (
    <aside className="flex flex-col gap-5 rounded-lg border border-slate-200 bg-white px-5 py-4 text-sm dark:border-gray-700 dark:bg-gray-900">
      <MetadataItem label="Category">
        <Select value={entry.category} disabled={disabled} onValueChange={onChangeCategory}>
          <SelectTrigger aria-label="Category" className="h-8 w-full">
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
      </MetadataItem>
      <MetadataItem label="Key">
        <code className="font-mono text-slate-800 dark:text-gray-100">{entry.key}</code>
      </MetadataItem>
      <MetadataItem label="Implementation">
        {entry.implemented ? (
          <span className="text-emerald-700 dark:text-emerald-300">Available in code</span>
        ) : (
          <span className="text-slate-600 dark:text-gray-300">Not implemented. Status stays Planned.</span>
        )}
      </MetadataItem>
      <MetadataItem label="Last change">{lastChange}</MetadataItem>
      <MetadataItem label="Created">{formatDate(entry.created_at)}</MetadataItem>
    </aside>
  );
}

function MetadataItem({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[11px] font-semibold tracking-wide text-slate-400 uppercase dark:text-gray-500">
        {label}
      </span>
      <div className="text-slate-700 dark:text-gray-200">{children}</div>
    </div>
  );
}

interface EntryMenuProps {
  entry: AdminIntakeEntry;
  onRename: () => void;
  onDelete: () => void;
}

export function EntryMenu({ entry, onRename, onDelete }: EntryMenuProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="icon-sm" aria-label="More actions">
          <MoreHorizontal size={16} />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={onRename}>
          <Pencil size={14} />
          Rename
        </DropdownMenuItem>
        {entry.deletable ? (
          <DropdownMenuItem onSelect={onDelete} className="text-red-600 focus:text-red-700">
            <Trash2 size={14} />
            Delete
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface RenameDialogProps {
  open: boolean;
  name: string;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onRename: (name: string) => void;
}

export function RenameDialog({ open, name, pending, onOpenChange, onRename }: RenameDialogProps) {
  const [value, setValue] = useState(name);

  useEffect(() => {
    if (open) setValue(name);
  }, [open, name]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <form
          className="flex flex-col gap-4"
          onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            onRename(value.trim());
          }}
        >
          <DialogHeader>
            <DialogTitle>Rename intake</DialogTitle>
            <DialogDescription>Companies see this name in the factory.</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="rename-intake">Name</Label>
            <Input
              id="rename-intake"
              value={value}
              onChange={(event: React.ChangeEvent<HTMLInputElement>) => setValue(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!value.trim() || pending}>
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface DeleteDialogProps {
  open: boolean;
  name: string;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onDelete: () => void;
}

export function DeleteDialog({ open, name, pending, onOpenChange, onDelete }: DeleteDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Delete {name}?</DialogTitle>
          <DialogDescription>This removes the intake from the catalog. You cannot undo this.</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" disabled={pending} onClick={onDelete}>
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
