/**
 * Read-only YAML preview for a canvas.
 *
 * Renders the canvas definition as syntax-highlighted YAML in a Monaco editor
 * with copy, download, and import actions. The editor is always read-only.
 */

import Editor from "@monaco-editor/react";
import { useState } from "react";
import { Copy, Download, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ImportYamlIntoCanvasDialog } from "./ImportYamlIntoCanvasDialog";

interface CanvasYamlViewProps {
  yamlText: string;
  filename: string;
  onCopy?: () => void;
  onDownload?: () => void;
  onImport?: (data: { nodes: unknown[]; edges: unknown[] }) => Promise<void>;
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
              <Button variant="outline" size="sm" onClick={() => setIsImportDialogOpen(true)}>
                <Upload />
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
            <Button variant="outline" size="sm" onClick={onCopy}>
              <Copy />
              Copy
            </Button>
          )}
          {onDownload && (
            <Button variant="outline" size="sm" onClick={onDownload}>
              <Download />
              Download
            </Button>
          )}
        </div>
      </div>
      <div className="canvas-yaml-monaco h-full min-h-0 min-w-0">
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
            renderLineHighlight: "line",
            renderLineHighlightOnlyWhenFocus: false,
          }}
        />
      </div>
    </div>
  );
}
