/**
 * Read-only YAML preview for a canvas.
 *
 * Renders the canvas definition as syntax-highlighted YAML in a Monaco editor
 * with copy and download actions. The editor is always read-only.
 */

import Editor from "@monaco-editor/react";
import { Copy, Download } from "lucide-react";
import { Button } from "@/components/ui/button";

interface CanvasYamlViewProps {
  yamlText: string;
  filename: string;
  onCopy?: () => void;
  onDownload?: () => void;
}

export function CanvasYamlView({ yamlText, filename, onCopy, onDownload }: CanvasYamlViewProps) {
  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-2">
        <span className="font-mono text-sm text-gray-600">{filename}</span>
        <div className="flex items-center gap-2">
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
