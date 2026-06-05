import { Editor } from "@monaco-editor/react";
import { useCallback, useEffect, useRef } from "react";

import { getWorkflowFileMonacoLanguage } from "./lib/workflow-monaco-language";

const fileEditorOptions = {
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: "on" as const,
  wordWrap: "on" as const,
  folding: true,
  automaticLayout: true,
  scrollBeyondLastLine: false,
  renderWhitespace: "boundary" as const,
  smoothScrolling: true,
  tabSize: 2,
  insertSpaces: true,
  cursorBlinking: "smooth" as const,
  contextmenu: true,
  selectOnLineNumbers: true,
  renderLineHighlight: "line" as const,
};

interface WorkflowFileMonacoEditorProps {
  path: string;
  content: string;
  language?: string;
  readOnly: boolean;
  onChange: (value: string) => void;
}

export function WorkflowFileMonacoEditor({
  path,
  content,
  language,
  readOnly,
  onChange,
}: WorkflowFileMonacoEditorProps) {
  const suppressNextChangeRef = useRef(false);
  const previousPathRef = useRef(path);

  useEffect(() => {
    if (previousPathRef.current === path) return;

    previousPathRef.current = path;
    suppressNextChangeRef.current = true;
  }, [path]);

  const handleChange = useCallback(
    (value: string | undefined) => {
      if (suppressNextChangeRef.current) {
        suppressNextChangeRef.current = false;
        return;
      }

      onChange(value ?? "");
    },
    [onChange],
  );

  return (
    <div className="min-h-0 flex-1 bg-white" data-testid="workflow-file-editor">
      <Editor
        key={path}
        height="100%"
        language={language ?? getWorkflowFileMonacoLanguage(path)}
        value={content}
        theme="vs"
        onChange={handleChange}
        options={{
          ...fileEditorOptions,
          readOnly,
          domReadOnly: readOnly,
        }}
      />
    </div>
  );
}
