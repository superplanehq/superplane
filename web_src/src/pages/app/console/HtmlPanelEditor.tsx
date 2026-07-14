import { forwardRef, useImperativeHandle, useMemo, useRef, useState, type RefObject } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import Editor, { type Monaco } from "@monaco-editor/react";
import type { editor as MonacoEditor } from "monaco-editor";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useResponsiveRailCollapse } from "@/hooks/useResponsiveRailCollapse";
import { cn } from "@/lib/utils";
import { useTheme } from "@/contexts/useTheme";

import { HtmlBody, HtmlBodyLoading } from "./HtmlBody";
import { markdownTextIsLoading } from "./markdownInterpolation";
import { MarkdownVariablesPanel } from "./MarkdownVariablesPanel";
import { useMarkdownVariables } from "./useMarkdownVariables";
import type { MarkdownVariable } from "./panelTypes";

/**
 * Imperative handle exposed by the body code editor to its parent. We don't
 * expose the raw Monaco editor; instead the panel card gets the two narrow
 * operations it needs (focus when the user clicks "Edit", insert a
 * `{{ variable.field }}` snippet at the caret from the variables rail) and
 * nothing else. That keeps the rest of the editor markup framework-agnostic
 * if we ever swap Monaco out.
 */
export interface HtmlCodeEditorHandle {
  focus(): void;
  insertAtCursor(snippet: string): void;
}

/**
 * In-card editor for an HTML panel. Mirrors the markdown editor: title input,
 * body textarea, collapsible live preview, and the variables manager on the
 * right rail. The body is sanitized inside `HtmlBody`, so the preview shows
 * exactly what saving would produce.
 *
 * Reuses `MarkdownVariablesPanel` and `useMarkdownVariables` because the
 * variable system is identical for markdown and html panels.
 */
function useHtmlEditorPreview(
  canvasId: string,
  draftTitle: string,
  draftBody: string,
  draftVariables: MarkdownVariable[],
) {
  const textForSideload = useMemo(() => `${draftTitle}\n${draftBody}`, [draftTitle, draftBody]);
  const { vars, errors, isLoading, baseLoading, sideloadLoading, searchingNames } = useMarkdownVariables(
    canvasId,
    draftVariables,
    textForSideload,
  );
  const previewLoading = markdownTextIsLoading(draftBody, baseLoading, sideloadLoading, searchingNames);
  return { previewVars: vars, errors, isLoading, previewLoading };
}

interface HtmlPanelEditorProps {
  panelId: string;
  canvasId: string;
  draftTitle: string;
  setDraftTitle: (value: string) => void;
  draftBody: string;
  setDraftBody: (value: string) => void;
  draftVariables: MarkdownVariable[];
  setDraftVariables: (next: MarkdownVariable[]) => void;
  titleInputRef: RefObject<HTMLInputElement | null>;
  codeEditorRef: RefObject<HtmlCodeEditorHandle | null>;
  /** Set when the last save attempt was blocked by variable validation. */
  saveError?: string | null;
  onCancel: () => void;
  onCommit: () => void;
}

export function HtmlPanelEditor({
  panelId,
  canvasId,
  draftTitle,
  setDraftTitle,
  draftBody,
  setDraftBody,
  draftVariables,
  setDraftVariables,
  titleInputRef,
  codeEditorRef,
  saveError,
  onCancel,
  onCommit,
}: HtmlPanelEditorProps) {
  const handleTitleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      onCancel();
      return;
    }
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      onCommit();
    }
  };

  const { previewVars, errors, isLoading, previewLoading } = useHtmlEditorPreview(
    canvasId,
    draftTitle,
    draftBody,
    draftVariables,
  );

  const [previewCollapsed, setPreviewCollapsed] = useState(false);

  // Variables rail collapses automatically when the parent widget is narrow.
  // The manual toggle wins until the breakpoint flips again, at which point
  // the auto behavior takes over so resizes always honor the current width.
  const {
    containerRef: gridRef,
    collapsed: variablesCollapsed,
    toggle: toggleVariablesCollapsed,
  } = useResponsiveRailCollapse();

  return (
    <div className="flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900">
      <div className="flex items-center gap-2 rounded-t-lg px-2 py-1">
        <Input
          ref={titleInputRef}
          value={draftTitle}
          onChange={(e) => setDraftTitle(e.target.value)}
          onKeyDown={handleTitleKeyDown}
          placeholder={panelId}
          aria-label="Panel title"
          className="h-7 border-0 bg-transparent px-2 text-xs font-medium text-slate-800 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0 dark:text-gray-100"
          data-testid="console-html-title-editor"
        />
      </div>
      <div
        ref={gridRef}
        className={cn(
          "grid min-h-0 flex-1 grid-cols-1 gap-0",
          variablesCollapsed ? "grid-cols-[minmax(0,1fr)_auto]" : "grid-cols-[minmax(0,1fr)_minmax(220px,320px)]",
        )}
      >
        <div className="flex min-h-0 min-w-0 flex-col border-r border-slate-950/10 dark:border-gray-800">
          <HtmlCodeEditor
            ref={codeEditorRef}
            value={draftBody}
            onChange={setDraftBody}
            onSave={onCommit}
            onCancel={onCancel}
          />
          <HtmlLivePreview
            body={draftBody}
            vars={previewVars}
            loading={previewLoading}
            collapsed={previewCollapsed}
            onToggle={() => setPreviewCollapsed((prev) => !prev)}
          />
        </div>
        <MarkdownVariablesPanel
          canvasId={canvasId}
          draftBody={draftBody}
          draftVariables={draftVariables}
          setDraftVariables={setDraftVariables}
          previewVars={previewVars}
          errors={errors}
          isLoading={isLoading}
          onInsertSnippet={(snippet) => codeEditorRef.current?.insertAtCursor(snippet)}
          collapsed={variablesCollapsed}
          onToggleCollapsed={toggleVariablesCollapsed}
        />
      </div>
      <HtmlEditorFooter saveError={saveError} onCancel={onCancel} onCommit={onCommit} />
    </div>
  );
}

function HtmlEditorFooter({
  saveError,
  onCancel,
  onCommit,
}: {
  saveError?: string | null;
  onCancel: () => void;
  onCommit: () => void;
}) {
  return (
    <div className="flex items-center justify-between gap-2 rounded-b-lg border-t border-slate-950/10 bg-slate-50/50 px-3 py-1.5 dark:border-gray-800 dark:bg-gray-800/50">
      {saveError ? (
        <span className="text-[11px] text-red-600 dark:text-red-400" role="alert" data-testid="console-html-save-error">
          {saveError}
        </span>
      ) : (
        <span className="text-[11px] text-slate-500 dark:text-gray-400">
          <kbd className="rounded border border-slate-200 bg-white px-1 font-mono dark:border-gray-600 dark:bg-gray-900">
            Esc
          </kbd>{" "}
          cancel &middot;{" "}
          <kbd className="rounded border border-slate-200 bg-white px-1 font-mono dark:border-gray-600 dark:bg-gray-900">
            Cmd+Enter
          </kbd>{" "}
          save
        </span>
      )}
      <div className="flex items-center gap-1">
        <Button type="button" size="sm" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="button" size="sm" onClick={onCommit} data-testid="console-html-save">
          Save
        </Button>
      </div>
    </div>
  );
}

/**
 * Monaco-backed HTML editor for the panel body. Wraps `@monaco-editor/react`
 * with the keybindings and focus/insert helpers the panel card needs.
 *
 * - Cmd/Ctrl+Enter saves; Escape cancels (matches the rest of the in-card
 *   edit affordances). Both are wired through `addCommand`, with the Escape
 *   command guarded by a "when" clause so it doesn't pre-empt Monaco's
 *   built-in widgets (suggest list, find box, rename input, …).
 * - The imperative handle exposes `focus()` for "Edit body" entry and
 *   `insertAtCursor()` for the variables panel's "Insert" buttons. Both
 *   tolerate the editor not being mounted yet by deferring the call to
 *   `onMount`, which matters because Monaco loads its module asynchronously
 *   and may not be ready in the same tick as the parent's focus effect.
 */
interface HtmlCodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  onSave: () => void;
  onCancel: () => void;
}

const CODE_EDITOR_OPTIONS: MonacoEditor.IStandaloneEditorConstructionOptions = {
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: "on",
  wordWrap: "on",
  scrollBeyondLastLine: false,
  smoothScrolling: true,
  tabSize: 2,
  automaticLayout: true,
  renderLineHighlight: "line",
  glyphMargin: false,
  folding: true,
  lineNumbersMinChars: 3,
  formatOnType: false,
  formatOnPaste: false,
};

const HtmlCodeEditor = forwardRef<HtmlCodeEditorHandle, HtmlCodeEditorProps>(function HtmlCodeEditor(
  { value, onChange, onSave, onCancel },
  ref,
) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";
  const editorRef = useRef<MonacoEditor.IStandaloneCodeEditor | null>(null);
  const pendingFocusRef = useRef(false);
  // Hold the latest callbacks in refs so the Monaco mount handler can call
  // through without re-binding commands on every parent render.
  const onSaveRef = useRef(onSave);
  const onCancelRef = useRef(onCancel);
  onSaveRef.current = onSave;
  onCancelRef.current = onCancel;

  useImperativeHandle(
    ref,
    () => ({
      focus() {
        if (editorRef.current) editorRef.current.focus();
        else pendingFocusRef.current = true;
      },
      insertAtCursor(snippet) {
        const editor = editorRef.current;
        const selection = editor?.getSelection();
        if (!editor || !selection) {
          onChange(value + snippet);
          return;
        }
        editor.executeEdits("console-html-insert-variable", [
          { range: selection, text: snippet, forceMoveMarkers: true },
        ]);
        editor.focus();
      },
    }),
    [onChange, value],
  );

  const handleMount = (instance: MonacoEditor.IStandaloneCodeEditor, monaco: Monaco) => {
    editorRef.current = instance;
    instance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => onSaveRef.current());
    instance.addCommand(
      monaco.KeyCode.Escape,
      () => onCancelRef.current(),
      "!suggestWidgetVisible && !findWidgetVisible && !renameInputVisible && !parameterHintsVisible",
    );
    if (pendingFocusRef.current) {
      pendingFocusRef.current = false;
      instance.focus();
    }
  };

  return (
    <div className="min-h-[160px] flex-1 overflow-hidden bg-white dark:bg-gray-900" data-testid="console-html-editor">
      <Editor
        height="100%"
        language="html"
        value={value}
        onChange={(next) => onChange(next ?? "")}
        onMount={handleMount}
        theme={monacoTheme}
        options={CODE_EDITOR_OPTIONS}
      />
    </div>
  );
});

function HtmlLivePreview({
  body,
  vars,
  loading,
  collapsed,
  onToggle,
}: {
  body: string;
  vars: Record<string, unknown>;
  loading: boolean;
  collapsed: boolean;
  onToggle: () => void;
}) {
  return (
    <div
      className={cn(
        "flex flex-col overflow-hidden border-t border-slate-950/10 bg-slate-50/40 dark:border-gray-800 dark:bg-gray-800/40",
        collapsed ? "shrink-0" : "min-h-[120px] flex-1",
      )}
      data-testid="console-html-editor-preview"
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={!collapsed}
        className="flex items-center justify-between gap-2 border-b border-slate-950/10 px-3 py-1.5 text-left hover:bg-slate-100/60 dark:border-gray-800 dark:hover:bg-gray-800/60"
        data-testid="console-html-editor-preview-toggle"
      >
        <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
          Preview
        </span>
        {collapsed ? (
          <ChevronUp className="size-3.5 text-slate-500 dark:text-gray-400" />
        ) : (
          <ChevronDown className="size-3.5 text-slate-500 dark:text-gray-400" />
        )}
      </button>
      {!collapsed ? (
        <div className="min-h-0 flex-1 overflow-auto px-3 py-2">
          {body.trim() ? (
            loading ? (
              <HtmlBodyLoading />
            ) : (
              <HtmlBody body={body} vars={vars} />
            )
          ) : (
            <p className="text-[12px] text-slate-400 dark:text-gray-500">
              Preview will appear once you write HTML above.
            </p>
          )}
        </div>
      ) : null}
    </div>
  );
}
