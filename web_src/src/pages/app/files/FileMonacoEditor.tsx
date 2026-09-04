import { Editor, type OnMount } from "@monaco-editor/react";
import { useCallback, useEffect, useRef } from "react";

import { useTheme } from "@/contexts/useTheme";
import { getFileMonacoLanguage } from "./lib/monaco-language";
import type { FileChangeStatus } from "./types";

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
  status?: FileChangeStatus;
  readOnly: boolean;
  onChange: (value: string) => void;
}

type MonacoEditor = Parameters<OnMount>[0];
type Monaco = Parameters<OnMount>[1];
type FileDecorationState = {
  path: string;
  content: string;
  status?: FileChangeStatus;
};

function buildFileContentDecorations(editor: MonacoEditor, monaco: Monaco, status?: FileChangeStatus) {
  const inlineClassName =
    status === "added"
      ? "!text-green-600 dark:!text-green-400"
      : status === "deleted"
        ? "!text-red-600 dark:!text-red-400"
        : undefined;
  if (!inlineClassName) return [];

  const model = editor.getModel();
  if (!model) return [];

  const lastLineNumber = model.getLineCount();
  return [
    {
      range: new monaco.Range(1, 1, lastLineNumber, model.getLineMaxColumn(lastLineNumber)),
      options: {
        inlineClassName,
      },
    },
  ];
}

export function FileMonacoEditor({ path, content, language, status, readOnly, onChange }: FileMonacoEditorProps) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";
  const suppressNextChangeRef = useRef(false);
  const previousPathRef = useRef(path);
  const editorRef = useRef<MonacoEditor | null>(null);
  const monacoRef = useRef<Monaco | null>(null);
  const decorationIdsRef = useRef<string[]>([]);
  const currentFileStateRef = useRef<FileDecorationState>({ path, content, status });
  const decoratedFileStateRef = useRef<FileDecorationState | null>(null);
  currentFileStateRef.current = { path, content, status };

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

  const applyFileDecorations = useCallback((editor: MonacoEditor, monaco: Monaco) => {
    const nextState = currentFileStateRef.current;
    const previousState = decoratedFileStateRef.current;
    if (
      previousState?.path === nextState.path &&
      previousState.content === nextState.content &&
      previousState.status === nextState.status
    ) {
      return;
    }

    decorationIdsRef.current = editor.deltaDecorations(
      decorationIdsRef.current,
      buildFileContentDecorations(editor, monaco, nextState.status),
    );
    decoratedFileStateRef.current = nextState;
  }, []);

  const handleMount = useCallback<OnMount>(
    (editor, monaco) => {
      editorRef.current = editor;
      monacoRef.current = monaco;
      applyFileDecorations(editor, monaco);
      editor.onDidDispose(() => {
        if (editorRef.current !== editor) return;
        editorRef.current = null;
        monacoRef.current = null;
        decorationIdsRef.current = [];
        decoratedFileStateRef.current = null;
      });
    },
    [applyFileDecorations],
  );

  useEffect(() => {
    const editor = editorRef.current;
    const monaco = monacoRef.current;
    if (!editor || !monaco) return;

    applyFileDecorations(editor, monaco);
  }, [applyFileDecorations, content, path, status]);

  return (
    <div className="min-h-0 flex-1 bg-white dark:bg-gray-900" data-testid="file-editor">
      <Editor
        key={path}
        height="100%"
        language={language ?? getFileMonacoLanguage(path)}
        value={content}
        theme={monacoTheme}
        onChange={handleChange}
        onMount={handleMount}
        options={{
          ...fileEditorOptions,
          readOnly,
          domReadOnly: readOnly,
        }}
      />
    </div>
  );
}
