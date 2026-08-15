import { Editor } from "@monaco-editor/react";
import { useMemo } from "react";
import type { CanvasesResolvedReplayInputStatus } from "@/api-client";
import { computeLineDiff } from "@/lib/jsonPayloadDiff";
import { cn } from "@/lib/utils";

const editorOptions = {
  minimap: { enabled: false },
  fontSize: 12,
  lineNumbers: "on" as const,
  wordWrap: "on" as const,
  folding: true,
  automaticLayout: true,
  scrollBeyondLastLine: false,
  tabSize: 2,
  insertSpaces: true,
};

const statusBadges: Partial<
  Record<CanvasesResolvedReplayInputStatus, { label: string; slug: string; className: string }>
> = {
  STATUS_DETACHED: {
    label: "Detached — no longer feeds this node",
    slug: "detached",
    className: "bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300",
  },
  STATUS_EXPIRED: {
    label: "Expired — original payload erased by retention",
    slug: "expired",
    className: "bg-rose-100 text-rose-800 dark:bg-rose-950/40 dark:text-rose-300",
  },
};

function ReplayPayloadDiff({ diffLines, testId }: { diffLines: ReturnType<typeof computeLineDiff>; testId: string }) {
  const hasChanges = diffLines.some((line) => line.type !== "unchanged");

  return (
    <div className="flex min-h-0 flex-col">
      <div className="mb-1 text-xs font-semibold text-gray-500">Diff vs original</div>
      <div
        className="min-h-0 flex-1 overflow-auto rounded border bg-white p-1 font-mono text-xs dark:bg-gray-900"
        data-testid={testId}
      >
        {!hasChanges && <div className="p-2 text-gray-400">No changes yet</div>}
        {diffLines
          .filter((line) => line.type !== "unchanged")
          .map((line, index) => (
            <div
              key={`${line.type}-${index}`}
              data-testid={`${testId}-line-${line.type}`}
              className={cn(
                "whitespace-pre-wrap px-1",
                line.type === "added" && "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40",
                line.type === "removed" && "bg-red-50 text-red-700 dark:bg-red-950/40",
              )}
            >
              {line.type === "added" ? "+ " : "- "}
              {line.text}
            </div>
          ))}
      </div>
    </div>
  );
}

export interface ReplayInputEditorProps {
  // "replay" for a single editor, "replay-input-N" once there are several — keeps
  // the single-input testids (replay-payload-editor, replay-diff, ...) stable.
  testIdPrefix: string;
  label: string;
  editedText: string;
  originalText: string;
  monacoTheme: string;
  status?: CanvasesResolvedReplayInputStatus;
  readOnly: boolean;
  error?: string | null;
  errorKind?: "invalid-json" | "missing" | null;
  onChange: (value: string) => void;
}

export function ReplayInputEditor({
  testIdPrefix,
  label,
  editedText,
  originalText,
  monacoTheme,
  status,
  readOnly,
  error,
  errorKind,
  onChange,
}: ReplayInputEditorProps) {
  const diffLines = useMemo(() => computeLineDiff(originalText, editedText), [originalText, editedText]);
  const options = useMemo(() => ({ ...editorOptions, readOnly }), [readOnly]);
  const badge = status ? statusBadges[status] : undefined;

  return (
    <div className="grid min-h-0 flex-1 grid-cols-2 gap-3">
      <div className="flex min-h-0 flex-col">
        <div className="mb-1 flex items-center gap-2 text-xs font-semibold text-gray-500">
          <span>{label}</span>
          {badge && (
            <span
              className={cn("rounded px-1.5 py-0.5 text-[10px] font-semibold", badge.className)}
              data-testid={`${testIdPrefix}-${badge.slug}-badge`}
            >
              {badge.label}
            </span>
          )}
        </div>
        <div className="min-h-0 flex-1 rounded border" data-testid={`${testIdPrefix}-payload-editor`}>
          <Editor
            height="100%"
            language="json"
            value={editedText}
            theme={monacoTheme}
            onChange={(value) => onChange(value ?? "")}
            options={options}
          />
        </div>
        {error && (
          <div
            className="mt-1 text-xs text-red-600"
            data-testid={`${testIdPrefix}-${errorKind === "missing" ? "missing" : "json"}-error`}
          >
            {error}
          </div>
        )}
      </div>

      <ReplayPayloadDiff diffLines={diffLines} testId={`${testIdPrefix}-diff`} />
    </div>
  );
}
