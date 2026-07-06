import { useMemo } from "react";
import { ChevronLeft, ChevronRight, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

import {
  MARKDOWN_VARIABLE_NAME_RE,
  type MarkdownMemoryVariableSource,
  type MarkdownRunVariableSource,
  type MarkdownVariable,
  type MarkdownVariableSource,
} from "./panelTypes";
import type { MarkdownVariableError } from "./useMarkdownVariables";
import { MemorySourceControls, RunSourceControls } from "./MarkdownVariableSourceControls";
import { VariablePreview } from "./VariablePreview";

type SourceKind = MarkdownVariableSource["kind"];

const SOURCE_OPTIONS: Array<{ value: SourceKind; label: string }> = [
  { value: "memory", label: "Memory record" },
  { value: "run", label: "Run" },
];

/** Default source seeded when the author clicks "Add variable". */
function defaultMemorySource(): MarkdownMemoryVariableSource {
  return { kind: "memory", namespace: "" };
}

function defaultRunSource(): MarkdownRunVariableSource {
  return { kind: "run", select: "latest" };
}

/**
 * Pick a fresh `varN` name when adding a variable, choosing the smallest free
 * suffix so reorderings and deletions don't leak hole-y names into the YAML.
 */
function nextVariableName(taken: Set<string>): string {
  for (let i = 1; i < 100; i += 1) {
    const candidate = `var${i}`;
    if (!taken.has(candidate)) return candidate;
  }
  return `var${Date.now()}`;
}

/**
 * Right-rail editor for markdown variables. Renders one row per variable
 * with a name input, source picker, source-specific controls, and an inline
 * preview of the resolved object. The bottom of the panel shows a quick
 * "insert" affordance for each known variable field so authors don't have
 * to type the `{{ ... }}` syntax by hand.
 */
export function MarkdownVariablesPanel({
  canvasId,
  draftBody: _draftBody,
  draftVariables,
  setDraftVariables,
  previewVars,
  errors,
  isLoading,
  onInsertSnippet,
  collapsed = false,
  onToggleCollapsed,
}: {
  canvasId: string;
  draftBody: string;
  draftVariables: MarkdownVariable[];
  setDraftVariables: (next: MarkdownVariable[]) => void;
  previewVars: Record<string, unknown>;
  errors: MarkdownVariableError[];
  isLoading: boolean;
  onInsertSnippet: (snippet: string) => void;
  /** When `true`, render the slim collapsed strip instead of the full rail. */
  collapsed?: boolean;
  /** Toggle the collapsed state. Required when `collapsed` is used. */
  onToggleCollapsed?: () => void;
}) {
  void _draftBody;
  const updateVariable = (index: number, next: MarkdownVariable) => {
    const out = draftVariables.slice();
    out[index] = next;
    setDraftVariables(out);
  };
  const removeVariable = (index: number) => {
    setDraftVariables(draftVariables.filter((_, i) => i !== index));
  };
  const addVariable = () => {
    const taken = new Set(draftVariables.map((v) => v.name));
    setDraftVariables([...draftVariables, { name: nextVariableName(taken), source: defaultMemorySource() }]);
  };

  const errorByName = useMemo(() => {
    const map = new Map<string, string>();
    for (const error of errors) {
      if (error.name) map.set(error.name, error.message);
    }
    return map;
  }, [errors]);

  // Names that appear more than once. On save `normalizeDraftVariables` keeps
  // only the first entry per name, so we flag duplicates here to warn the
  // author before they lose the shadowed rows' configuration.
  const duplicateNames = useMemo(() => {
    const counts = new Map<string, number>();
    for (const variable of draftVariables) {
      const name = variable.name?.trim();
      if (!name) continue;
      counts.set(name, (counts.get(name) ?? 0) + 1);
    }
    const dups = new Set<string>();
    for (const [name, count] of counts) {
      if (count > 1) dups.add(name);
    }
    return dups;
  }, [draftVariables]);

  if (collapsed) {
    return <CollapsedVariablesStrip count={draftVariables.length} onToggleCollapsed={onToggleCollapsed} />;
  }

  return (
    <div
      // `min-h-0 min-w-0` ensures the panel can shrink to its grid track even
      // when its scrolling children carry intrinsic min-content widths. The
      // scroll region below owns both axis overflows so the column itself
      // never grows beyond the parent grid track.
      className="flex min-h-0 min-w-0 flex-col bg-slate-50/40 text-xs text-slate-700"
      data-testid="console-markdown-variables"
    >
      <div className="flex items-center justify-between gap-1 border-b border-slate-950/10 px-3 py-2 dark:border-gray-800/70">
        <div className="flex min-w-0 items-center gap-1">
          {onToggleCollapsed ? (
            <Button
              type="button"
              size="icon"
              variant="ghost"
              onClick={onToggleCollapsed}
              aria-label="Collapse variables"
              className="h-6 w-6 shrink-0 text-slate-500 hover:text-slate-700"
              data-testid="console-markdown-variables-collapse"
            >
              <ChevronRight className="size-3.5" />
            </Button>
          ) : null}
          <Label className="truncate text-xs font-semibold uppercase tracking-wide text-slate-600">Variables</Label>
        </div>
        <Button type="button" size="sm" variant="outline" className="h-7 shrink-0 gap-1" onClick={addVariable}>
          <Plus className="size-3.5" />
          Add
        </Button>
      </div>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-x-auto overflow-y-auto px-3 py-3">
        {draftVariables.length === 0 ? (
          <p className="rounded border border-dashed border-slate-300 bg-white px-3 py-4 text-center text-[12px] text-slate-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400">
            No variables yet. Add one to reference live data with{" "}
            <code className="rounded bg-slate-100 px-1 py-0.5">{"{{ name.field }}"}</code>.
          </p>
        ) : (
          draftVariables.map((variable, index) => (
            <VariableRow
              key={index}
              canvasId={canvasId}
              variable={variable}
              error={errorByName.get(variable.name)}
              duplicate={duplicateNames.has(variable.name?.trim())}
              previewValue={previewVars[variable.name]}
              onChange={(next) => updateVariable(index, next)}
              onRemove={() => removeVariable(index)}
              onInsertSnippet={onInsertSnippet}
              loading={isLoading}
            />
          ))
        )}
      </div>
    </div>
  );
}

/**
 * Slim vertical strip shown in place of the full variables rail when the
 * panel is collapsed (either auto-collapsed because the parent widget is
 * narrow, or because the author explicitly hid the rail). Clicking anywhere
 * on the strip re-expands the rail; the count badge is just an
 * at-a-glance affordance so the author knows whether there are variables to
 * come back to.
 */
function CollapsedVariablesStrip({ count, onToggleCollapsed }: { count: number; onToggleCollapsed?: () => void }) {
  return (
    <button
      type="button"
      onClick={onToggleCollapsed}
      aria-label="Expand variables"
      aria-expanded={false}
      className="flex h-full w-9 shrink-0 flex-col items-center gap-2 border-l border-slate-950/10 bg-slate-50/40 py-2 text-slate-500 hover:bg-slate-100/60 hover:text-slate-700 dark:border-gray-800/70 dark:bg-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
      data-testid="console-markdown-variables-expand"
    >
      <ChevronLeft className="size-3.5" />
      <span
        className="text-[10px] font-semibold uppercase tracking-wide"
        style={{ writingMode: "vertical-rl", transform: "rotate(180deg)" }}
      >
        Variables
      </span>
      {count > 0 ? (
        <span className="mt-1 rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium text-slate-600">
          {count}
        </span>
      ) : null}
    </button>
  );
}

function VariableRow({
  canvasId,
  variable,
  error,
  duplicate,
  previewValue,
  onChange,
  onRemove,
  onInsertSnippet,
  loading,
}: {
  canvasId: string;
  variable: MarkdownVariable;
  error?: string;
  duplicate?: boolean;
  previewValue: unknown;
  onChange: (next: MarkdownVariable) => void;
  onRemove: () => void;
  onInsertSnippet: (snippet: string) => void;
  loading: boolean;
}) {
  const sourceKind: SourceKind = variable.source?.kind === "run" ? "run" : "memory";
  const nameIsValid = MARKDOWN_VARIABLE_NAME_RE.test(variable.name);

  const setSourceKind = (kind: SourceKind) => {
    if (kind === sourceKind) return;
    onChange({ ...variable, source: kind === "memory" ? defaultMemorySource() : defaultRunSource() });
  };

  return (
    // `min-w-0` lets the card collapse to its container, so the Input below
    // (which is intrinsically as wide as its placeholder) doesn't push the
    // row past the column boundary.
    <div className="min-w-0 space-y-2 rounded-md border border-slate-200 bg-white p-3 shadow-sm dark:border-gray-800/70 dark:bg-gray-900 dark:shadow-none">
      <div className="flex min-w-0 items-center gap-2">
        <Input
          value={variable.name}
          onChange={(e) => onChange({ ...variable, name: e.target.value })}
          placeholder="name"
          aria-label="Variable name"
          className={cn(
            "h-7 min-w-0 flex-1 font-mono text-[12px]",
            ((!nameIsValid && variable.name) || duplicate) && "border-red-400 focus-visible:ring-red-300",
          )}
          data-testid="markdown-variable-name"
        />
        <Select value={sourceKind} onValueChange={(value) => setSourceKind(value as SourceKind)}>
          <SelectTrigger className="h-7 w-[120px] shrink-0 text-[12px]" aria-label="Variable source kind">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {SOURCE_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          type="button"
          size="icon"
          variant="ghost"
          onClick={onRemove}
          aria-label="Remove variable"
          className="h-7 w-7 shrink-0 text-slate-500 hover:bg-red-50 hover:text-red-600"
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>
      {!nameIsValid && variable.name ? (
        <p className="text-[11px] text-red-600">Use letters, digits, and underscore. Don&apos;t start with a digit.</p>
      ) : null}
      {nameIsValid && duplicate ? (
        <p className="text-[11px] text-red-600">
          Duplicate name. Only the first variable with this name is kept on save.
        </p>
      ) : null}

      {variable.source?.kind === "memory" ? (
        <MemorySourceControls
          canvasId={canvasId}
          source={variable.source}
          onChange={(next) => onChange({ ...variable, source: next })}
        />
      ) : null}
      {variable.source?.kind === "run" ? (
        <RunSourceControls source={variable.source} onChange={(next) => onChange({ ...variable, source: next })} />
      ) : null}

      <VariablePreview
        name={variable.name}
        value={previewValue}
        error={error}
        loading={loading}
        onInsertSnippet={onInsertSnippet}
      />
    </div>
  );
}
