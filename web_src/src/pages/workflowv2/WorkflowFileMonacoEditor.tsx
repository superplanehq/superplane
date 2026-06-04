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
  const contentRef = useRef(content);
  contentRef.current = content;

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

      const next = value ?? "";
      // Ignore echoes from programmatic `value` updates: when the controlled
      // content prop changes (e.g. staged content reloads), Monaco fires onChange
      // with the same value we just set. Propagating it would be treated as a user
      // edit and could wrongly unstage a file whose content matches its baseline.
      if (next === contentRef.current) {
        return;
      }

      onChange(next);
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
