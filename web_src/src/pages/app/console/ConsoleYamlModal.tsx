import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Editor } from "@monaco-editor/react";
import { AlertCircle, Copy, Download, Upload } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { consoleToYaml, consoleYamlFilename, parseConsoleYaml } from "./consoleYaml";

export type ConsoleYamlModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  panels: ConsolePanel[];
  layout: ConsoleLayoutItem[];
  canvasId?: string;
  canvasName?: string;
  /** When provided, the modal allows importing YAML. When omitted, it is view-only. */
  onImport?: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => Promise<void>;
  isImporting?: boolean;
};

export function ConsoleYamlModal({
  open,
  onOpenChange,
  panels,
  layout,
  canvasId,
  canvasName,
  onImport,
  isImporting,
}: ConsoleYamlModalProps) {
  const exportedYaml = useMemo(
    () => consoleToYaml({ panels, layout, canvasId, canvasName }),
    [panels, layout, canvasId, canvasName],
  );
  const filename = useMemo(() => consoleYamlFilename(canvasName), [canvasName]);
  const editor = useConsoleYamlEditor({ open, exportedYaml, filename, onImport, onOpenChange });
  const isDirty = editor.text !== exportedYaml;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="large" className="flex max-h-[90vh] w-[90vw] h-full flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="border-b border-slate-200 px-4 py-3">
          <DialogTitle className="flex items-center gap-2 text-sm font-mono text-slate-600">{filename}</DialogTitle>
          <DialogDescription className="sr-only">
            View, copy, download, or import the console as a YAML file. Imports replace every panel and layout entry.
          </DialogDescription>
        </DialogHeader>

        <ConsoleYamlToolbar
          isDirty={isDirty}
          canImport={Boolean(onImport)}
          fileInputRef={editor.fileInputRef}
          onFileUpload={editor.handleFileUpload}
          onCopy={editor.handleCopy}
          onDownload={editor.handleDownload}
        />

        <ConsoleYamlEditor text={editor.text} readOnly={!onImport} onChange={editor.handleEditorChange} />

        <ConsoleYamlError message={editor.parseError} />

        <ConsoleYamlFooter
          canImport={Boolean(onImport)}
          isDirty={isDirty}
          isImporting={Boolean(isImporting)}
          onClose={() => onOpenChange(false)}
          onApply={editor.handleApplyClick}
        />
      </DialogContent>

      <ReplaceConsoleDialog
        open={editor.confirmingReplace}
        isImporting={Boolean(isImporting)}
        onOpenChange={editor.setConfirmingReplace}
        onConfirm={editor.handleConfirmReplace}
      />
    </Dialog>
  );
}

function useConsoleYamlEditor({
  open,
  exportedYaml,
  filename,
  onImport,
  onOpenChange,
}: {
  open: boolean;
  exportedYaml: string;
  filename: string;
  onImport?: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => Promise<void>;
  onOpenChange: (open: boolean) => void;
}) {
  const [text, setText] = useState(exportedYaml);
  const [parseError, setParseError] = useState<string | null>(null);
  const [confirmingReplace, setConfirmingReplace] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (!open) return;
    setText(exportedYaml);
    setParseError(null);
  }, [open, exportedYaml]);

  const handleEditorChange = useCallback(
    (value: string | undefined) => {
      setText(value ?? "");
      if (parseError) setParseError(null);
    },
    [parseError],
  );

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      showSuccessToast("Console YAML copied to clipboard");
    } catch {
      showErrorToast("Failed to copy YAML to clipboard");
    }
  }, [text]);

  const handleDownload = useCallback(() => {
    downloadYaml(text, filename);
    showSuccessToast("Console exported as YAML");
  }, [text, filename]);

  const handleFileUpload = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!isYamlFile(file)) {
      setParseError("Please select a .yaml or .yml file.");
      return;
    }
    readYamlFile(file, setText, setParseError);
    if (fileInputRef.current) fileInputRef.current.value = "";
  }, []);

  const handleApplyClick = useCallback(() => {
    setParseError(null);
    const result = parseConsoleYaml(text);
    if (!result.ok) {
      setParseError(result.error);
      return;
    }
    setConfirmingReplace(true);
  }, [text]);

  const handleConfirmReplace = useCallback(async () => {
    if (!onImport) return;
    const result = parseConsoleYaml(text);
    if (!result.ok) {
      setParseError(result.error);
      setConfirmingReplace(false);
      return;
    }
    try {
      await onImport({ panels: result.data.spec.panels, layout: result.data.spec.layout });
      setConfirmingReplace(false);
      onOpenChange(false);
      showSuccessToast("Console imported from YAML");
    } catch (e) {
      setConfirmingReplace(false);
      setParseError(e instanceof Error ? e.message : "Failed to import console.");
    }
  }, [text, onImport, onOpenChange]);

  return {
    text,
    parseError,
    confirmingReplace,
    setConfirmingReplace,
    fileInputRef,
    handleEditorChange,
    handleCopy,
    handleDownload,
    handleFileUpload,
    handleApplyClick,
    handleConfirmReplace,
  };
}

function downloadYaml(text: string, filename: string) {
  const blob = new Blob([text], { type: "text/yaml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function isYamlFile(file: File): boolean {
  return file.name.endsWith(".yaml") || file.name.endsWith(".yml");
}

function readYamlFile(file: File, setText: (text: string) => void, setParseError: (message: string | null) => void) {
  const reader = new FileReader();
  reader.onload = (event) => {
    const result = event.target?.result;
    if (typeof result === "string") {
      setText(result);
      setParseError(null);
    }
  };
  reader.onerror = () => setParseError("Failed to read file.");
  reader.readAsText(file);
}

function ConsoleYamlEditor({
  text,
  readOnly,
  onChange,
}: {
  text: string;
  readOnly: boolean;
  onChange: (value: string | undefined) => void;
}) {
  return (
    <div className="flex-1 min-h-0">
      <Editor
        height="100%"
        language="yaml"
        value={text}
        onChange={onChange}
        theme="vs"
        options={editorOptions(readOnly)}
      />
    </div>
  );
}

function editorOptions(readOnly: boolean) {
  return {
    readOnly,
    domReadOnly: readOnly,
    minimap: { enabled: false },
    fontSize: 13,
    lineNumbers: "on" as const,
    wordWrap: "on" as const,
    folding: true,
    scrollBeyondLastLine: false,
    renderWhitespace: "boundary" as const,
    smoothScrolling: true,
    tabSize: 2,
    renderLineHighlight: "line" as const,
  };
}

function ConsoleYamlToolbar({
  isDirty,
  canImport,
  fileInputRef,
  onFileUpload,
  onCopy,
  onDownload,
}: {
  isDirty: boolean;
  canImport: boolean;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  onFileUpload: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onCopy: () => void;
  onDownload: () => void;
}) {
  return (
    <div className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-2">
      <span className="text-xs text-slate-500">
        {isDirty ? "Editor has unsaved YAML edits" : "Showing live console YAML"}
      </span>
      <div className="flex items-center gap-2">
        {canImport ? (
          <>
            <input
              ref={fileInputRef}
              type="file"
              accept=".yaml,.yml"
              hidden
              onChange={onFileUpload}
              data-testid="console-yaml-file-input"
            />
            <Button
              variant="outline"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
              data-testid="console-yaml-upload"
            >
              <Upload className="mr-1 h-3.5 w-3.5" />
              Upload
            </Button>
          </>
        ) : null}
        <Button variant="outline" size="sm" onClick={onCopy} data-testid="console-yaml-copy">
          <Copy className="mr-1 h-3.5 w-3.5" />
          Copy
        </Button>
        <Button variant="outline" size="sm" onClick={onDownload} data-testid="console-yaml-download">
          <Download className="mr-1 h-3.5 w-3.5" />
          Download
        </Button>
      </div>
    </div>
  );
}

function ConsoleYamlError({ message }: { message: string | null }) {
  if (!message) return null;

  return (
    <div
      className="flex items-start gap-2 border-t border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700"
      data-testid="console-yaml-error"
    >
      <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function ConsoleYamlFooter({
  canImport,
  isDirty,
  isImporting,
  onClose,
  onApply,
}: {
  canImport: boolean;
  isDirty: boolean;
  isImporting: boolean;
  onClose: () => void;
  onApply: () => void;
}) {
  return (
    <DialogFooter className="border-t border-slate-200 px-4 py-3">
      <Button variant="outline" onClick={onClose}>
        Close
      </Button>
      {canImport ? (
        <Button onClick={onApply} disabled={!isDirty || isImporting} data-testid="console-yaml-apply">
          {isImporting ? "Applying…" : "Apply YAML"}
        </Button>
      ) : null}
    </DialogFooter>
  );
}

function ReplaceConsoleDialog({
  open,
  isImporting,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  isImporting: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Replace console?</DialogTitle>
          <DialogDescription>
            Applying this YAML replaces every panel and layout entry for the current console. This action cannot be
            undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={isImporting}
            data-testid="console-yaml-replace-confirm"
          >
            Replace console
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
