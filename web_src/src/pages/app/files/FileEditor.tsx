import { lazy, Suspense } from "react";

const FileMonacoEditor = lazy(() =>
  import("./FileMonacoEditor").then((module) => ({ default: module.FileMonacoEditor })),
);

export function FileEditor({
  path,
  content,
  deleted,
  language,
  loading,
  errorMessage,
  disabled,
  onChange,
}: {
  path: string | null;
  content: string;
  deleted: boolean;
  language?: string;
  loading: boolean;
  errorMessage?: string;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  if (!path) {
    return <div className="min-h-0 flex-1 bg-white" />;
  }

  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500">Loading file...</div>
    );
  }

  if (errorMessage) {
    return <div className="p-4 text-sm text-red-600">{errorMessage}</div>;
  }

  if (deleted) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500">
        File marked for deletion
      </div>
    );
  }

  return (
    <Suspense
      fallback={
        <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-slate-500">Loading editor...</div>
      }
    >
      <FileMonacoEditor path={path} content={content} language={language} readOnly={disabled} onChange={onChange} />
    </Suspense>
  );
}
