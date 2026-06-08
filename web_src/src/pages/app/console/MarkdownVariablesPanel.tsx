import { useMemo } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

import {
  MARKDOWN_RUN_SELECTS,
  MARKDOWN_VARIABLE_DIRECTIONS,
  MARKDOWN_VARIABLE_NAME_RE,
  type MarkdownMemoryVariableSource,
  type MarkdownRunSelect,
  type MarkdownRunVariableSource,
  type MarkdownVariable,
  type MarkdownVariableDirection,
  type MarkdownVariableMatch,
  type MarkdownVariableSource,
} from "./panelTypes";
import type { MarkdownVariableError } from "./useMarkdownVariables";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";
import { VariablePreview } from "./VariablePreview";

type SourceKind = MarkdownVariableSource["kind"];

const SOURCE_OPTIONS: Array<{ value: SourceKind; label: string }> = [
  { value: "memory", label: "Memory record" },
  { value: "run", label: "Run" },
];

const RUN_SELECT_LABELS: Record<MarkdownRunSelect, string> = {
  latest: "Latest run",
  latest_passed: "Latest passed run",
  latest_failed: "Latest failed run",
};

const DIRECTION_LABELS: Record<MarkdownVariableDirection, string> = {
  desc: "Descending",
  asc: "Ascending",
};

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
}: {
  canvasId: string;
  draftBody: string;
  draftVariables: MarkdownVariable[];
  setDraftVariables: (next: MarkdownVariable[]) => void;
  previewVars: Record<string, unknown>;
  errors: MarkdownVariableError[];
  isLoading: boolean;
  onInsertSnippet: (snippet: string) => void;
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

  return (
    <div
      // `min-h-0 min-w-0` ensures the panel can shrink to its grid track even
      // when its scrolling children carry intrinsic min-content widths. The
      // scroll region below owns both axis overflows so the column itself
      // never grows beyond the parent grid track.
      className="flex min-h-0 min-w-0 flex-col bg-slate-50/40 text-xs text-slate-700"
      data-testid="console-markdown-variables"
    >
      <div className="flex items-center justify-between border-b border-slate-950/10 px-3 py-2">
        <Label className="text-xs font-semibold uppercase tracking-wide text-slate-600">Variables</Label>
        <Button type="button" size="sm" variant="outline" className="h-7 gap-1" onClick={addVariable}>
          <Plus className="size-3.5" />
          Add
        </Button>
      </div>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-x-auto overflow-y-auto px-3 py-3">
        {draftVariables.length === 0 ? (
          <p className="rounded border border-dashed border-slate-300 bg-white px-3 py-4 text-center text-[12px] text-slate-500">
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
    <div className="min-w-0 space-y-2 rounded-md border border-slate-200 bg-white p-3 shadow-sm">
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

function MemorySourceControls({
  canvasId,
  source,
  onChange,
}: {
  canvasId: string;
  source: MarkdownMemoryVariableSource;
  onChange: (next: MarkdownMemoryVariableSource) => void;
}) {
  const { namespaces, fields } = useMemoryCatalog(canvasId, source.namespace);
  const orderByOptions = useMemo(() => {
    const set = new Set<string>(["createdAt", "updatedAt"]);
    for (const field of fields) set.add(field.field);
    if (source.orderBy?.trim()) set.add(source.orderBy.trim());
    return Array.from(set);
  }, [fields, source.orderBy]);

  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <Label className="text-[11px] font-medium text-slate-600">Namespace</Label>
        <Select
          value={source.namespace || undefined}
          onValueChange={(value) => onChange({ ...source, namespace: value })}
        >
          <SelectTrigger className="h-7 text-[12px]" data-testid="markdown-variable-memory-namespace">
            <SelectValue placeholder="Select a namespace" />
          </SelectTrigger>
          <SelectContent>
            {namespaces.length === 0 ? (
              <SelectItem value="__empty__" disabled>
                No namespaces in this canvas
              </SelectItem>
            ) : (
              namespaces.map((ns) => (
                <SelectItem key={ns.namespace} value={ns.namespace}>
                  {ns.namespace}
                  <span className="ml-1 text-[11px] text-slate-400">({ns.count})</span>
                </SelectItem>
              ))
            )}
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <Label className="text-[11px] font-medium text-slate-600">Order by</Label>
          <Select
            value={source.orderBy || "createdAt"}
            onValueChange={(value) => onChange({ ...source, orderBy: value })}
          >
            <SelectTrigger className="h-7 text-[12px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {orderByOptions.map((option) => (
                <SelectItem key={option} value={option}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-[11px] font-medium text-slate-600">Direction</Label>
          <Select
            value={source.direction ?? "desc"}
            onValueChange={(value) => onChange({ ...source, direction: value as MarkdownVariableDirection })}
          >
            <SelectTrigger className="h-7 text-[12px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MARKDOWN_VARIABLE_DIRECTIONS.map((dir) => (
                <SelectItem key={dir} value={dir}>
                  {DIRECTION_LABELS[dir]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <MemoryMatchesEditor matches={source.matches ?? []} onChange={(next) => onChange({ ...source, matches: next })} />
    </div>
  );
}

function MemoryMatchesEditor({
  matches,
  onChange,
}: {
  matches: MarkdownVariableMatch[];
  onChange: (next: MarkdownVariableMatch[]) => void;
}) {
  const updateMatch = (index: number, next: MarkdownVariableMatch) => {
    const out = matches.slice();
    out[index] = next;
    onChange(out);
  };
  const addMatch = () => onChange([...matches, { field: "", value: "" }]);
  const removeMatch = (index: number) => onChange(matches.filter((_, i) => i !== index));

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between">
        <Label className="text-[11px] font-medium text-slate-600">Match (field equals value)</Label>
        <Button type="button" size="sm" variant="outline" className="h-6 gap-1" onClick={addMatch}>
          <Plus className="size-3" />
          Add match
        </Button>
      </div>
      {matches.length === 0 ? (
        <p className="text-[11px] text-slate-400">Optional. Add a match to pick a specific row.</p>
      ) : (
        <div className="space-y-1.5">
          {matches.map((match, index) => (
            <div key={index} className="flex min-w-0 items-center gap-1">
              <Input
                value={match.field}
                onChange={(e) => updateMatch(index, { ...match, field: e.target.value })}
                placeholder="field"
                className="h-7 min-w-0 flex-1 text-[12px]"
                aria-label="Match field"
              />
              <Input
                value={match.value}
                onChange={(e) => updateMatch(index, { ...match, value: e.target.value })}
                placeholder="value"
                className="h-7 min-w-0 flex-1 text-[12px]"
                aria-label="Match value"
              />
              <Button
                type="button"
                size="icon"
                variant="ghost"
                onClick={() => removeMatch(index)}
                aria-label="Remove match"
                className="h-7 w-7 shrink-0 text-slate-500 hover:bg-red-50 hover:text-red-600"
              >
                <Trash2 className="size-3" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function RunSourceControls({
  source,
  onChange,
}: {
  source: MarkdownRunVariableSource;
  onChange: (next: MarkdownRunVariableSource) => void;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-[11px] font-medium text-slate-600">Run</Label>
      <Select
        value={source.select}
        onValueChange={(value) => onChange({ ...source, select: value as MarkdownRunSelect })}
      >
        <SelectTrigger className="h-7 text-[12px]" data-testid="markdown-variable-run-select">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {MARKDOWN_RUN_SELECTS.map((select) => (
            <SelectItem key={select} value={select}>
              {RUN_SELECT_LABELS[select]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
