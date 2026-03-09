import Editor from "@monaco-editor/react";
import { Copy, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as yaml from "js-yaml";

interface CanvasYamlViewProps {
  yamlText: string;
  filename: string;
  readOnly?: boolean;
  onCopy?: () => void;
  onDownload?: () => void;
  onChange?: (parsed: { metadata?: Record<string, unknown>; spec?: Record<string, unknown> }) => void;
}

export function CanvasYamlView({ yamlText, filename, readOnly, onCopy, onDownload, onChange }: CanvasYamlViewProps) {
  const [editedText, setEditedText] = useState(yamlText);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    setEditedText(yamlText);
  }, [yamlText]);

  const parseError = useMemo(() => {
    if (editedText === yamlText) return null;
    try {
      yaml.load(editedText);
      return null;
    } catch (e) {
      return e instanceof Error ? e.message : "Invalid YAML";
    }
  }, [editedText, yamlText]);

  const handleEditorChange = useCallback((value: string | undefined) => {
    const text = value ?? "";
    setEditedText(text);
    try {
      const parsed = yaml.load(text);
      if (parsed && typeof parsed === "object") {
        onChangeRef.current?.(parsed as { metadata?: Record<string, unknown>; spec?: Record<string, unknown> });
      }
    } catch {
      // parse error -- shown via parseError memo
    }
  }, []);

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
      {parseError && (
        <div className="border-b border-red-200 bg-red-50 px-4 py-1.5 text-xs text-red-600">{parseError}</div>
      )}
      <div className="flex-1">
        <Editor
          height="100%"
          language="yaml"
          value={editedText}
          onChange={readOnly ? undefined : handleEditorChange}
          theme="vs"
          options={{
            readOnly: !!readOnly,
            domReadOnly: !!readOnly,
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
