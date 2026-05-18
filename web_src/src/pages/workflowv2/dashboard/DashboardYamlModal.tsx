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
import type { DashboardLayoutItem, DashboardPanel } from "@/hooks/useCanvasData";

import { dashboardToYaml, dashboardYamlFilename, parseDashboardYaml } from "./dashboardYaml";

export type DashboardYamlModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  panels: DashboardPanel[];
  layout: DashboardLayoutItem[];
  canvasId?: string;
  canvasName?: string;
  /** When provided, the modal allows importing YAML. When omitted, it is view-only. */
  onImport?: (next: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => Promise<void>;
  isImporting?: boolean;
};

export function DashboardYamlModal({
  open,
  onOpenChange,
  panels,
  layout,
  canvasId,
  canvasName,
  onImport,
  isImporting,
}: DashboardYamlModalProps) {
  const exportedYaml = useMemo(
    () => dashboardToYaml({ panels, layout, canvasId, canvasName }),
    [panels, layout, canvasId, canvasName],
  );
  const filename = useMemo(() => dashboardYamlFilename(canvasName), [canvasName]);

  const [editorText, setEditorText] = useState(exportedYaml);
  const [parseError, setParseError] = useState<string | null>(null);
  const [confirmingReplace, setConfirmingReplace] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  // Keep editor in sync with the latest exported YAML when the modal opens
  // or the underlying dashboard changes from outside.
  useEffect(() => {
    if (open) {
      setEditorText(exportedYaml);
      setParseError(null);
    }
  }, [open, exportedYaml]);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(editorText);
      showSuccessToast("Dashboard YAML copied to clipboard");
    } catch {
      showErrorToast("Failed to copy YAML to clipboard");
    }
  }, [editorText]);

  const handleDownload = useCallback(() => {
    const blob = new Blob([editorText], { type: "text/yaml;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    showSuccessToast("Dashboard exported as YAML");
  }, [editorText, filename]);

  const handleFileUpload = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.name.endsWith(".yaml") && !file.name.endsWith(".yml")) {
      setParseError("Please select a .yaml or .yml file.");
      return;
    }
    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result;
      if (typeof text === "string") {
        setEditorText(text);
        setParseError(null);
      }
    };
    reader.onerror = () => setParseError("Failed to read file.");
    reader.readAsText(file);

    if (fileInputRef.current) fileInputRef.current.value = "";
  }, []);

  const handleApplyClick = useCallback(() => {
    setParseError(null);
    const result = parseDashboardYaml(editorText);
    if (!result.ok) {
      setParseError(result.error);
      return;
    }
    setConfirmingReplace(true);
  }, [editorText]);

  const handleConfirmReplace = useCallback(async () => {
    if (!onImport) return;
    const result = parseDashboardYaml(editorText);
    if (!result.ok) {
      setParseError(result.error);
      setConfirmingReplace(false);
      return;
    }
    try {
      await onImport({ panels: result.data.spec.panels, layout: result.data.spec.layout });
      setConfirmingReplace(false);
      onOpenChange(false);
      showSuccessToast("Dashboard imported from YAML");
    } catch (e) {
      setConfirmingReplace(false);
      setParseError(e instanceof Error ? e.message : "Failed to import dashboard.");
    }
  }, [editorText, onImport, onOpenChange]);

  const isDirty = editorText !== exportedYaml;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="large" className="flex max-h-[90vh] w-[90vw] h-full flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="border-b border-slate-200 px-4 py-3">
          <DialogTitle className="flex items-center gap-2 text-sm font-mono text-slate-600">{filename}</DialogTitle>
          <DialogDescription className="sr-only">
            View, copy, download, or import the dashboard as a YAML file. Imports replace every panel and layout entry.
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-2">
          <span className="text-xs text-slate-500">
            {isDirty ? "Editor has unsaved YAML edits" : "Showing live dashboard YAML"}
          </span>
          <div className="flex items-center gap-2">
            {onImport ? (
              <>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".yaml,.yml"
                  hidden
                  onChange={handleFileUpload}
                  data-testid="dashboard-yaml-file-input"
                />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => fileInputRef.current?.click()}
                  data-testid="dashboard-yaml-upload"
                >
                  <Upload className="mr-1 h-3.5 w-3.5" />
                  Upload
                </Button>
              </>
            ) : null}
            <Button variant="outline" size="sm" onClick={handleCopy} data-testid="dashboard-yaml-copy">
              <Copy className="mr-1 h-3.5 w-3.5" />
              Copy
            </Button>
            <Button variant="outline" size="sm" onClick={handleDownload} data-testid="dashboard-yaml-download">
              <Download className="mr-1 h-3.5 w-3.5" />
              Download
            </Button>
          </div>
        </div>

        <div className="flex-1 min-h-0">
          <Editor
            height="100%"
            language="yaml"
            value={editorText}
            onChange={(value) => {
              setEditorText(value ?? "");
              if (parseError) setParseError(null);
            }}
            theme="vs"
            options={{
              readOnly: !onImport,
              domReadOnly: !onImport,
              minimap: { enabled: false },
              fontSize: 13,
              lineNumbers: "on",
              wordWrap: "on",
              folding: true,
              scrollBeyondLastLine: false,
              renderWhitespace: "boundary",
              smoothScrolling: true,
              tabSize: 2,
              renderLineHighlight: "line",
            }}
          />
        </div>

        {parseError ? (
          <div
            className="flex items-start gap-2 border-t border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700"
            data-testid="dashboard-yaml-error"
          >
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>{parseError}</span>
          </div>
        ) : null}

        <DialogFooter className="border-t border-slate-200 px-4 py-3">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          {onImport ? (
            <Button onClick={handleApplyClick} disabled={!isDirty || isImporting} data-testid="dashboard-yaml-apply">
              {isImporting ? "Applying…" : "Apply YAML"}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>

      <Dialog open={confirmingReplace} onOpenChange={setConfirmingReplace}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Replace dashboard?</DialogTitle>
            <DialogDescription>
              Applying this YAML replaces every panel and layout entry for the current dashboard. This action cannot be
              undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmingReplace(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmReplace}
              disabled={isImporting}
              data-testid="dashboard-yaml-replace-confirm"
            >
              Replace dashboard
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  );
}
