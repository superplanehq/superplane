import { useCallback, useRef, useState } from "react";
import { Upload, FileText, AlertCircle } from "lucide-react";
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
import { useCreateCanvas } from "@/hooks/useCanvasData";
import { showErrorToast, showSuccessToast } from "@/utils/toast";

interface ImportYamlDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  onSuccess: (canvasId: string) => void;
}

interface ParsedCanvas {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    description?: string;
    isTemplate?: boolean;
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

  if (!parsed.metadata?.name) {
    return "Canvas metadata.name is required.";
  }

  return null;
}

export function ImportYamlDialog({ open, onOpenChange, organizationId, onSuccess }: ImportYamlDialogProps) {
  const [yamlText, setYamlText] = useState("");
  const [parseError, setParseError] = useState<string | null>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const createMutation = useCreateCanvas(organizationId);

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

    try {
      const result = await createMutation.mutateAsync({
        name: parsed.metadata!.name!,
        description: parsed.metadata?.description,
        nodes: (parsed.spec?.nodes as any[]) || [],
        edges: (parsed.spec?.edges as any[]) || [],
      });

      const canvasId = result?.data?.canvas?.metadata?.id;
      if (canvasId) {
        showSuccessToast("Canvas imported successfully");
        handleOpenChange(false);
        onSuccess(canvasId);
      }
    } catch (error) {
      const errorMessage = (error as Error)?.message || "Failed to import canvas";
      showErrorToast(errorMessage);
    }
  }, [yamlText, createMutation, handleOpenChange, onSuccess]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Import Canvas from YAML</DialogTitle>
          <DialogDescription>Upload a YAML file or paste a Canvas definition to create a new Canvas.</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label htmlFor="yaml-file-input" className="mb-2">
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
                id="yaml-file-input"
                data-testid="import-yaml-file-input"
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
            <Label htmlFor="yaml-paste-input" className="mb-2">
              YAML definition
            </Label>
            <Textarea
              id="yaml-paste-input"
              data-testid="import-yaml-textarea"
              value={yamlText}
              onChange={(e) => {
                setYamlText(e.target.value);
                setParseError(null);
                setFileName(null);
              }}
              placeholder={`apiVersion: v1\nkind: Canvas\nmetadata:\n  name: my-canvas\nspec:\n  nodes: []\n  edges: []`}
              rows={12}
              className="font-mono text-sm"
            />
          </div>

          {parseError && (
            <div
              data-testid="import-yaml-error"
              className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3"
            >
              <AlertCircle className="h-4 w-4 text-red-500 mt-0.5 shrink-0" />
              <span className="text-sm text-red-700">{parseError}</span>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            data-testid="import-yaml-submit"
            onClick={handleImport}
            disabled={!yamlText.trim() || createMutation.isPending}
          >
            {createMutation.isPending ? "Importing..." : "Import Canvas"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
