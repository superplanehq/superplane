import Editor from "@monaco-editor/react";
import type { Monaco } from "@monaco-editor/react";
import type { editor as MonacoEditor } from "monaco-editor";
import { Copy, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useCallback, useEffect, useRef, useState } from "react";
import * as yaml from "js-yaml";
import { validateCanvasYaml, type YamlDiagnostic } from "./yamlCanvasValidation";

interface CanvasYamlViewProps {
  yamlText: string;
  filename: string;
  readOnly?: boolean;
  serverError?: string | null;
  onCopy?: () => void;
  onDownload?: () => void;
  onChange?: (parsed: { metadata?: Record<string, unknown>; spec?: Record<string, unknown> }) => void;
}

const MARKER_OWNER = "canvas-validation";
const VALIDATION_DEBOUNCE_MS = 300;

function toMonacoSeverity(monaco: Monaco, severity: YamlDiagnostic["severity"]): number {
  return severity === "error" ? monaco.MarkerSeverity.Error : monaco.MarkerSeverity.Warning;
}

export function CanvasYamlView({
  yamlText,
  filename,
  readOnly,
  serverError,
  onCopy,
  onDownload,
  onChange,
}: CanvasYamlViewProps) {
  const [editedText, setEditedText] = useState(yamlText);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  const editorRef = useRef<MonacoEditor.IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<Monaco | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setEditedText(yamlText);
  }, [yamlText]);

  const runValidation = useCallback((text: string, extraServerError?: string | null) => {
    const monaco = monacoRef.current;
    const editor = editorRef.current;
    if (!monaco || !editor) return;

    const model = editor.getModel();
    if (!model) return;

    const markers: MonacoEditor.IMarkerData[] = [];

    try {
      const parsed = yaml.load(text);
      if (parsed && typeof parsed === "object") {
        const diagnostics = validateCanvasYaml(parsed as Record<string, unknown>, text);
        for (const d of diagnostics) {
          markers.push({
            startLineNumber: d.startLineNumber,
            endLineNumber: d.endLineNumber,
            startColumn: d.startColumn,
            endColumn: d.endColumn,
            message: d.message,
            severity: toMonacoSeverity(monaco, d.severity),
          });
        }
      }
    } catch (e) {
      const yamlError = e as yaml.YAMLException;
      const line = yamlError?.mark?.line != null ? yamlError.mark.line + 1 : 1;
      const col = yamlError?.mark?.column != null ? yamlError.mark.column + 1 : 1;
      markers.push({
        startLineNumber: line,
        endLineNumber: line,
        startColumn: col,
        endColumn: col + 1,
        message: yamlError?.reason || yamlError?.message || "Invalid YAML syntax",
        severity: monaco.MarkerSeverity.Error,
      });
    }

    if (extraServerError) {
      markers.push({
        startLineNumber: 1,
        endLineNumber: 1,
        startColumn: 1,
        endColumn: 2,
        message: `Server: ${extraServerError}`,
        severity: monaco.MarkerSeverity.Error,
      });
    }

    monaco.editor.setModelMarkers(model, MARKER_OWNER, markers);
  }, []);

  useEffect(() => {
    runValidation(editedText, serverError);
  }, [editedText, serverError, runValidation]);

  const handleEditorMount = useCallback(
    (editor: MonacoEditor.IStandaloneCodeEditor, monaco: Monaco) => {
      editorRef.current = editor;
      monacoRef.current = monaco;
      runValidation(editedText, serverError);
    },
    [editedText, serverError, runValidation],
  );

  const handleEditorChange = useCallback(
    (value: string | undefined) => {
      const text = value ?? "";
      setEditedText(text);

      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }

      debounceRef.current = setTimeout(() => {
        runValidation(text);

        try {
          const parsed = yaml.load(text);
          if (parsed && typeof parsed === "object") {
            const diagnostics = validateCanvasYaml(parsed as Record<string, unknown>, text);
            const hasErrors = diagnostics.some((d) => d.severity === "error");
            if (!hasErrors) {
              onChangeRef.current?.(parsed as { metadata?: Record<string, unknown>; spec?: Record<string, unknown> });
            }
          }
        } catch {
          // YAML parse error -- markers already set by runValidation
        }
      }, VALIDATION_DEBOUNCE_MS);
    },
    [runValidation],
  );

  useEffect(() => {
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
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
      <div className="flex-1">
        <Editor
          height="100%"
          language="yaml"
          value={editedText}
          onChange={readOnly ? undefined : handleEditorChange}
          onMount={handleEditorMount}
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
