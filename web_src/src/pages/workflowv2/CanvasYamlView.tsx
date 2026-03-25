/**
 * Read-only YAML preview for a canvas.
 *
 * Renders the canvas definition as syntax-highlighted YAML in a Monaco editor
 * with copy, download, and import actions. The editor is always read-only.
 */

import Editor from "@monaco-editor/react";
import { useCallback, useRef, useState } from "react";
import { AlertCircle, Copy, Download, FileText, Upload } from "lucide-react";
import * as yaml from "js-yaml";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface ParsedCanvas {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    description?: string;
  };
  spec?: {
    nodes?: unknown[];
    edges?: unknown[];
  };
}

function validateCanvasYaml(parsed: ParsedCanvas): string | null {
  if (parsed.apiVersion && parsed.apiVersion !== "v1") {
    return `Unsupported apiVersion "${parsed.apiVersion}". Only "v1" is supported.`;
  }

  if (parsed.kind && parsed.kind !== "Canvas") {
    return `Unsupported kind "${parsed.kind}". Only "Canvas" is supported.`;
  }

  return null;
}

interface ImportYamlDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onImport: (data: { nodes: unknown[]; edges: unknown[] }) => void;
  isImporting?: boolean;
}

function ImportYamlIntoCanvasDialog({ open, onOpenChange, onImport, isImporting }: ImportYamlDialogProps) {
  const [yamlText, setYamlText] = useState("");
  const [parseError, setParseError] = useState<string | null>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const reset = useCallback(() => {
    setYamlText("");
    setParseError(null);
    setFileName(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  }, []);

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        reset();
      }
      onOpenChange(nextOpen);
    },
    [onOpenChange, reset],
  );

  const handleFileSelect = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    if (!file.name.endsWith(".yaml") && !file.name.endsWith(".yml")) {
      setParseError("Please select a .yaml or .yml file.");
      return;
    }

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      setYamlText(content);
      setFileName(file.name);
      setParseError(null);
    };
    reader.onerror = () => {
      setParseError("Failed to read the file.");
    };
    reader.readAsText(file);
  }, []);

  const handleImport = useCallback(() => {
    setParseError(null);

    const trimmed = yamlText.trim();
    if (!trimmed) {
      setParseError("Please provide a YAML definition.");
      return;
    }

    let parsed: ParsedCanvas;
    try {
      parsed = yaml.load(trimmed) as ParsedCanvas;
    } catch (e) {
      const message = e instanceof Error ? e.message : "Unknown error";
      setParseError(`Invalid YAML syntax: ${message}`);
      return;
    }

    if (!parsed || typeof parsed !== "object") {
      setParseError("YAML content must be a valid object.");
      return;
    }

    const validationError = validateCanvasYaml(parsed);
    if (validationError) {
      setParseError(validationError);
      return;
    }

    onImport({
      nodes: (parsed.spec?.nodes as unknown[]) || [],
      edges: (parsed.spec?.edges as unknown[]) || [],
    });
    handleOpenChange(false);
  }, [yamlText, onImport, handleOpenChange]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Import YAML</DialogTitle>
          <DialogDescription>Upload a YAML file or paste a Canvas definition to update this canvas.</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label htmlFor="yaml-import-file-input" className="mb-2">
              Upload YAML file
            </Label>
            <div
              className="flex items-center gap-3 rounded-md border border-dashed border-gray-300 p-4 cursor-pointer hover:border-gray-400 transition-colors"
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="h-5 w-5 text-gray-400" />
              <div className="flex-1">
                {fileName ? (
                  <div className="flex items-center gap-2">
                    <FileText className="h-4 w-4 text-gray-500" />
                    <span className="text-sm text-gray-700">{fileName}</span>
                  </div>
                ) : (
                  <span className="text-sm text-gray-500">Click to select a .yaml or .yml file</span>
                )}
              </div>
              <input
                ref={fileInputRef}
                id="yaml-import-file-input"
                type="file"
                accept=".yaml,.yml"
                className="hidden"
                onChange={handleFileSelect}
              />
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="h-px flex-1 bg-gray-200" />
            <span className="text-xs text-gray-400">or paste YAML below</span>
            <div className="h-px flex-1 bg-gray-200" />
          </div>

          <div>
            <Label htmlFor="yaml-import-paste-input" className="mb-2">
              YAML definition
            </Label>
            <Textarea
              id="yaml-import-paste-input"
              value={yamlText}
              onChange={(e) => {
                setYamlText(e.target.value);
                setParseError(null);
                setFileName(null);
              }}
              placeholder={`apiVersion: v1\nkind: Canvas\nmetadata:\n  name: my-canvas\nspec:\n  nodes: []\n  edges: []`}
              rows={12}
              className="font-mono text-sm max-h-[50vh] overflow-y-auto"
            />
          </div>

          {parseError && (
            <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3">
              <AlertCircle className="h-4 w-4 text-red-500 mt-0.5 shrink-0" />
              <span className="text-sm text-red-700">{parseError}</span>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleImport} disabled={!yamlText.trim() || isImporting}>
            {isImporting ? "Importing..." : "Import"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface CanvasYamlViewProps {
  yamlText: string;
  filename: string;
  onCopy?: () => void;
  onDownload?: () => void;
  onImport?: (data: { nodes: unknown[]; edges: unknown[] }) => void;
  isImporting?: boolean;
}

export function CanvasYamlView({ yamlText, filename, onCopy, onDownload, onImport, isImporting }: CanvasYamlViewProps) {
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-2">
        <span className="font-mono text-sm text-gray-600">{filename}</span>
        <div className="flex items-center gap-2">
          {onImport && (
            <>
              <Button
                variant="outline"
                size="sm"
                className="h-8 gap-1.5 text-xs"
                onClick={() => setIsImportDialogOpen(true)}
              >
                <Upload className="h-3.5 w-3.5" />
                Import
              </Button>
              <ImportYamlIntoCanvasDialog
                open={isImportDialogOpen}
                onOpenChange={setIsImportDialogOpen}
                onImport={onImport}
                isImporting={isImporting}
              />
            </>
          )}
          {onCopy && (
            <Button variant="outline" size="sm" className="h-8 gap-1.5 text-xs" onClick={onCopy}>
              <Copy className="h-3.5 w-3.5" />
              Copy
            </Button>
          )}
          {onDownload && (
            <Button variant="outline" size="sm" className="h-8 gap-1.5 text-xs" onClick={onDownload}>
              <Download className="h-3.5 w-3.5" />
              Download
            </Button>
          )}
        </div>
      </div>
      <div className="flex-1">
        <Editor
          height="100%"
          language="yaml"
          value={yamlText}
          theme="vs"
          options={{
            readOnly: true,
            domReadOnly: true,
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: "on",
            wordWrap: "on",
            folding: true,
            scrollBeyondLastLine: false,
            renderWhitespace: "boundary",
            smoothScrolling: true,
            tabSize: 2,
          }}
        />
      </div>
    </div>
  );
}
