/**
 * Dialog for importing a YAML definition into an existing canvas.
 *
 * Accepts either a file upload (.yaml / .yml) or pasted text, validates the
 * content against the v1 Canvas schema, and hands the parsed nodes + edges
 * back to the caller via `onImport`.
 */

import { useCallback, useRef, useState } from "react";
import { AlertCircle, FileText, Upload } from "lucide-react";
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
  if (!parsed.spec || !Array.isArray(parsed.spec.nodes)) {
    return "YAML must contain a spec with a nodes array. Is this a valid Canvas definition?";
  }
  return null;
}

function parseCanvasYaml(text: string): { data: { nodes: unknown[]; edges: unknown[] } } | { error: string } {
  const trimmed = text.trim();
  if (!trimmed) return { error: "Please provide a YAML definition." };

  let parsed: ParsedCanvas;
  try {
    parsed = yaml.load(trimmed) as ParsedCanvas;
  } catch (e) {
    return { error: `Invalid YAML syntax: ${e instanceof Error ? e.message : "Unknown error"}` };
  }

  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { error: "YAML content must be a valid object." };
  }

  const validationError = validateCanvasYaml(parsed);
  if (validationError) return { error: validationError };

  return {
    data: {
      nodes: (parsed.spec?.nodes as unknown[]) || [],
      edges: (parsed.spec?.edges as unknown[]) || [],
    },
  };
}

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

  const handleImport = useCallback(async () => {
    setParseError(null);
    const result = parseCanvasYaml(yamlText);
    if ("error" in result) {
      setParseError(result.error);
      return;
    }
    try {
      await onImport(result.data);
      handleOpenChange(false);
    } catch (err) {
      setParseError(err instanceof Error ? err.message : "Import failed. Please try again.");
    }
  }, [yamlText, onImport, handleOpenChange]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[85vh] flex flex-col">
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
