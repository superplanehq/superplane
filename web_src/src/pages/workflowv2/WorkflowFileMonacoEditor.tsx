import { Editor } from "@monaco-editor/react";
import { useCallback, useEffect, useRef } from "react";

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
        language={language ?? getMonacoLanguage(path)}
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

function getMonacoLanguage(path: string): string {
  const normalizedPath = path.toLowerCase();
  const extension = normalizedPath.split(".").pop();

  if (normalizedPath.endsWith("dockerfile") || normalizedPath.includes("/dockerfile")) return "dockerfile";
  if (normalizedPath.endsWith("makefile") || normalizedPath.includes("/makefile")) return "makefile";

  switch (extension) {
    case "css":
      return "css";
    case "go":
      return "go";
    case "html":
      return "html";
    case "js":
    case "mjs":
    case "cjs":
      return "javascript";
    case "json":
    case "jsonc":
      return "json";
    case "jsx":
      return "javascript";
    case "md":
    case "mdx":
    case "markdown":
      return "markdown";
    case "py":
      return "python";
    case "sh":
    case "bash":
    case "zsh":
      return "shell";
    case "ts":
      return "typescript";
    case "tsx":
      return "typescript";
    case "xml":
      return "xml";
    case "yaml":
    case "yml":
      return "yaml";
    default:
      return "plaintext";
  }
}
