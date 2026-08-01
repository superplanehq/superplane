import { lazy, Suspense } from "react";

import { MarkdownContent } from "../Markdown";
import { getFileMonacoLanguage } from "./lib/monaco-language";
import type { FileChangeStatus } from "./types";

const FileMonacoEditor = lazy(() =>
  import("./FileMonacoEditor").then((module) => ({ default: module.FileMonacoEditor })),
);

export function FileEditor({
  path,
  content,
  canvasId,
  organizationId,
  status,
  language,
  loading,
  errorMessage,
  disabled,
  onChange,
}: {
  path: string | null;
  content: string;
  canvasId?: string;
  organizationId?: string;
  status?: FileChangeStatus;
  language?: string;
  loading: boolean;
  errorMessage?: string;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  if (!path) {
    return <div className="min-h-0 flex-1 bg-white dark:bg-gray-900" />;
  }

  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500 dark:text-gray-400">
        Loading file...
      </div>
    );
  }

  if (errorMessage) {
    return <div className="p-4 text-sm text-red-600">{errorMessage}</div>;
  }

  const resolvedLanguage = language ?? getFileMonacoLanguage(path);
  const isMarkdown = resolvedLanguage === "markdown";
  const deleted = status === "deleted";

  if (disabled && isMarkdown && !deleted) {
    return (
      <div className="min-h-0 flex-1 overflow-auto bg-white p-6 dark:bg-gray-900">
        <MarkdownContent
          content={content}
          canvasId={canvasId}
          organizationId={organizationId}
          data-testid="file-markdown-preview"
        />
      </div>
    );
  }

  return (
    <Suspense
      fallback={
        <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500 dark:text-gray-400">
          Loading editor...
        </div>
      }
    >
      <FileMonacoEditor
        path={path}
        content={content}
        language={language}
        status={status}
        readOnly={disabled}
        onChange={onChange}
      />
    </Suspense>
  );
}
