/**
 * Dialog for importing a YAML definition into an existing canvas.
 *
 * Accepts either a file upload (.yaml / .yml) or pasted text, validates the
 * content against the v1 Canvas schema, and hands the parsed nodes + edges
 * back to the caller via `onImport`.
 */

import { useCallback, useRef, useState } from "react";
import { analytics } from "@/lib/analytics";
import { AlertCircle, FileText, Upload } from "lucide-react";
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

import { parseCanvasYamlForImport } from "./lib/workflow-spec-files";

export interface ImportYamlDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onImport: (data: { nodes: unknown[]; edges: unknown[] }) => Promise<void>;
  isImporting?: boolean;
}

function ImportYamlFileUpload({
  fileName,
  fileInputRef,
  onFileSelect,
}: {
  fileName: string | null;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  onFileSelect: (event: React.ChangeEvent<HTMLInputElement>) => void;
}) {
  return (
    <div>
      <Label htmlFor="yaml-import-file-input" className="mb-2">
        Upload YAML file
      </Label>
      <div
        className="flex cursor-pointer items-center gap-3 rounded-md border border-dashed border-gray-300 p-4 transition-colors hover:border-gray-400"
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
          onChange={onFileSelect}
        />
      </div>
    </div>
  );
}

export function ImportYamlIntoCanvasDialog({ open, onOpenChange, onImport, isImporting }: ImportYamlDialogProps) {
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
    reader.onload = (event) => {
      const content = event.target?.result as string;
      setYamlText(content);
      setFileName(file.name);
      setParseError(null);
    };
    reader.onerror = () => {
      setParseError("Failed to read the file.");
    };
    reader.readAsText(file);
  }, []);

  const handleImport = useCallback(async () => {
    setParseError(null);
    const result = parseCanvasYamlForImport(yamlText);
    if (!result.ok) {
      setParseError(result.error);
      return;
    }

    try {
      await onImport({
        nodes: result.spec.nodes ?? [],
        edges: result.spec.edges ?? [],
      });
      analytics.yamlImport();
      handleOpenChange(false);
    } catch (error) {
      setParseError(error instanceof Error ? error.message : "Import failed. Please try again.");
    }
  }, [yamlText, onImport, handleOpenChange]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Import YAML</DialogTitle>
          <DialogDescription>Upload a YAML file or paste a Canvas definition to update this canvas.</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <ImportYamlFileUpload fileName={fileName} fileInputRef={fileInputRef} onFileSelect={handleFileSelect} />

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
              onChange={(event) => {
                setYamlText(event.target.value);
                setParseError(null);
                setFileName(null);
              }}
              placeholder={`apiVersion: v1\nkind: Canvas\nmetadata:\n  name: my-canvas\nspec:\n  nodes: []\n  edges: []`}
              rows={12}
              className="max-h-[50vh] overflow-y-auto font-mono text-sm"
            />
          </div>

          {parseError ? (
            <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-red-500" />
              <span className="text-sm text-red-700">{parseError}</span>
            </div>
          ) : null}
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
