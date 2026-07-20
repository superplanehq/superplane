import { Editor } from "@monaco-editor/react";
import { useCallback, useEffect, useRef } from "react";

import { useTheme } from "@/contexts/useTheme";
import { getFileMonacoLanguage } from "./lib/monaco-language";

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

interface FileMonacoEditorProps {
  path: string;
  content: string;
  language?: string;
  readOnly: boolean;
  onChange: (value: string) => void;
}

export function FileMonacoEditor({ path, content, language, readOnly, onChange }: FileMonacoEditorProps) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";
  const suppressNextChangeRef = useRef(false);
  const previousPathRef = useRef(path);

  useEffect(() => {
    if (previousPathRef.current === path) return;

    previousPathRef.current = path;
    suppressNextChangeRef.current = true;
  }, [path]);

  const handleChange = useCallback(
    (value: string | undefined) => {
      const next = value ?? "";
      if (suppressNextChangeRef.current) {
        suppressNextChangeRef.current = false;
        // Monaco often does not emit an onChange when the controlled value is
        // applied after a path switch, so the flag would otherwise swallow the
        // user's first real edit. Only ignore echoes of the current value.
        if (next === content) {
          return;
        }
      }

      onChange(next);
    },
    [content, onChange],
  );

  return (
    <div className="min-h-0 flex-1 bg-white dark:bg-gray-900" data-testid="file-editor">
      <Editor
        key={path}
        height="100%"
        language={language ?? getFileMonacoLanguage(path)}
        value={content}
        theme={monacoTheme}
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
