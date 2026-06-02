import { Editor } from "@monaco-editor/react";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const EDITOR_OPTIONS = {
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: "on" as const,
  wordWrap: "on" as const,
  folding: true,
  bracketPairColorization: { enabled: true },
  autoIndent: "advanced" as const,
  formatOnPaste: true,
  formatOnType: true,
  tabSize: 2,
  insertSpaces: true,
  scrollBeyondLastLine: false,
  smoothScrolling: true,
  cursorBlinking: "smooth" as const,
  renderLineHighlight: "line" as const,
  renderLineHighlightOnlyWhenFocus: false,
};

export type CanvasMemoryBankDialogMode = "create" | "edit";

interface CanvasMemoryBankDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: CanvasMemoryBankDialogMode;
  // For "edit" mode this is the existing bank's namespace; for "create" mode
  // it is undefined.
  originalNamespace?: string;
  initialEntries?: unknown[];
  isSubmitting?: boolean;
  onSubmit: (input: { namespace: string; entries: unknown[] }) => Promise<void>;
}

function stringifyEntries(entries: unknown[] | undefined): string {
  if (!entries || entries.length === 0) {
    return "[]";
  }
  try {
    return JSON.stringify(entries, null, 2);
  } catch {
    return "[]";
  }
}

export function CanvasMemoryBankDialog({
  open,
  onOpenChange,
  mode,
  originalNamespace,
  initialEntries,
  isSubmitting,
  onSubmit,
}: CanvasMemoryBankDialogProps) {
  const initialJson = useMemo(() => stringifyEntries(initialEntries), [initialEntries]);
  const [namespace, setNamespace] = useState<string>(originalNamespace ?? "");
  const [jsonValue, setJsonValue] = useState<string>(initialJson);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setNamespace(originalNamespace ?? "");
    setJsonValue(initialJson);
    setError(null);
  }, [open, originalNamespace, initialJson]);

  const title = mode === "create" ? "Create memory bank" : "Edit memory bank";
  const description =
    mode === "create"
      ? "Define a manually-managed memory bank by providing a namespace and a JSON array of entries."
      : "Edit the namespace or replace the entries of this manually-managed memory bank.";

  const handleSubmit = async () => {
    setError(null);

    const trimmedNamespace = namespace.trim();
    if (!trimmedNamespace) {
      setError("Namespace is required.");
      return;
    }

    let parsed: unknown;
    try {
      parsed = JSON.parse(jsonValue);
    } catch (e) {
      const message = e instanceof Error ? e.message : "Invalid JSON";
      setError(`Invalid JSON: ${message}`);
      return;
    }

    if (!Array.isArray(parsed)) {
      setError("Entries must be a JSON array.");
      return;
    }

    try {
      await onSubmit({ namespace: trimmedNamespace, entries: parsed });
      onOpenChange(false);
    } catch (e) {
      const message = e instanceof Error ? e.message : "Failed to save memory bank.";
      setError(message);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="large" className="flex h-[80vh] w-[90vw] max-w-3xl flex-col gap-0 overflow-hidden p-0">
        <div className="flex flex-col gap-1 border-b border-slate-950/10 px-6 py-4">
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-4 px-6 py-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="memory-bank-namespace">Namespace</Label>
            <Input
              id="memory-bank-namespace"
              value={namespace}
              onChange={(event) => setNamespace(event.target.value)}
              placeholder="e.g. release-cache"
              autoComplete="off"
              spellCheck={false}
              data-testid="memory-bank-namespace-input"
            />
          </div>

          <div className="flex min-h-0 flex-1 flex-col gap-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="memory-bank-entries">Entries (JSON array)</Label>
              <span className="text-xs text-gray-500">Each element becomes a row in the bank.</span>
            </div>
            <div
              id="memory-bank-entries"
              className="min-h-0 min-w-0 flex-1 overflow-hidden rounded-md border border-gray-300 bg-white dark:border-gray-700"
              data-testid="memory-bank-entries-editor"
            >
              <Editor
                height="100%"
                language="json"
                value={jsonValue}
                onChange={(value) => setJsonValue(value ?? "")}
                theme="vs"
                options={EDITOR_OPTIONS}
              />
            </div>
          </div>

          {error ? (
            <p className="text-xs text-red-600 dark:text-red-400" data-testid="memory-bank-dialog-error">
              {error}
            </p>
          ) : null}
        </div>

        <div className="flex justify-end gap-2 border-t border-slate-950/10 px-6 py-3">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? "Saving…" : mode === "create" ? "Create bank" : "Save changes"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
