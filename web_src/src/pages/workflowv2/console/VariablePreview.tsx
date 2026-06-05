import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";

/**
 * Show the resolved value for a variable as a compact field/value list with a
 * one-click "insert" button per field. Falls back to a short message when the
 * lookup found nothing yet (loading) or the source returned no data (error).
 */
export function VariablePreview({
  name,
  value,
  error,
  loading,
  onInsertSnippet,
}: {
  name: string;
  value: unknown;
  error?: string;
  loading: boolean;
  onInsertSnippet: (snippet: string) => void;
}) {
  if (!name.trim()) {
    return <p className="text-[11px] text-slate-400">Give this variable a name to enable the preview.</p>;
  }

  // `useMarkdownVariables` resolves in-flight variables to `null` (not
  // `undefined`) while their backing query loads, so gate on `== null` to cover
  // both. A non-null value during a background refetch still renders its fields
  // (stale-while-revalidate) instead of flashing the loading text.
  if (loading && value == null) {
    return <p className="text-[11px] text-slate-400">Loading preview…</p>;
  }

  if (error) {
    return (
      <p className="text-[11px] text-amber-600" data-testid="markdown-variable-preview-error">
        {error}
      </p>
    );
  }

  if (value == null) {
    return <p className="text-[11px] text-slate-400">No data resolved yet.</p>;
  }

  const fields = previewableFields(value);
  return <VariablePreviewBlock name={name} fields={fields} onInsertSnippet={onInsertSnippet} />;
}

/**
 * Inner collapsible body for `VariablePreview`. Pulled out so the collapsed
 * state hook can live on a real component instance (one per variable card)
 * without forcing the outer wrapper through all of its early-return cases.
 *
 * The card itself never widens: long preview values are clipped with
 * `truncate` and rendered inside a `min-w-0 flex-1` cell so flex's default
 * content-min-width doesn't push the card past the variables-panel column.
 */
function VariablePreviewBlock({
  name,
  fields,
  onInsertSnippet,
}: {
  name: string;
  fields: Array<{ key: string; preview: string }>;
  onInsertSnippet: (snippet: string) => void;
}) {
  const insertable = (suffix: string) => `{{ ${name}${suffix ? "." + suffix : ""} }}`;
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div className="min-w-0 space-y-1 rounded border border-slate-100 bg-slate-50 p-2">
      <div className="flex min-w-0 items-center gap-2 text-[11px] text-slate-500">
        <button
          type="button"
          onClick={() => setCollapsed((prev) => !prev)}
          aria-expanded={!collapsed}
          className="flex shrink-0 items-center gap-1 rounded px-1 py-0.5 hover:bg-slate-200/60"
          data-testid="markdown-variable-preview-toggle"
        >
          {collapsed ? <ChevronRight className="size-3" /> : <ChevronDown className="size-3" />}
          <span className="font-semibold uppercase tracking-wide">Preview</span>
        </button>
        <span className="min-w-0 flex-1" />
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-5 shrink-0 px-1.5 text-[11px]"
          onClick={() => onInsertSnippet(insertable(""))}
          title={`Insert ${insertable("")}`}
        >
          Insert
        </Button>
      </div>
      {!collapsed ? (
        fields.length === 0 ? (
          <p className="text-[11px] text-slate-500">Resolved value has no readable fields.</p>
        ) : (
          // The list can grow taller than the card on rich objects (memory
          // rows, runs with many `$["Node"]` accessors). Cap its height so
          // the variable card stays compact and let the list scroll
          // vertically inside the preview without affecting the outer
          // variables-panel scroll.
          <ul className="max-h-40 space-y-0.5 overflow-y-auto pr-1">
            {fields.map((field) => (
              <li key={field.key} className="flex min-w-0 items-start gap-2">
                <button
                  type="button"
                  onClick={() => onInsertSnippet(insertable(field.key))}
                  className="max-w-[55%] shrink-0 truncate text-left font-mono text-[11px] text-sky-700 underline-offset-2 hover:underline"
                  title={`Insert ${insertable(field.key)}`}
                >
                  {field.key}
                </button>
                <span className="min-w-0 flex-1 truncate text-[11px] text-slate-600" title={field.preview}>
                  {field.preview}
                </span>
              </li>
            ))}
          </ul>
        )
      ) : null}
    </div>
  );
}

/**
 * Pick a short list of (key, preview-string) pairs from a resolved variable
 * value. Object-shaped values (memory rows, run rows) get a flat list of
 * their own top-level keys. For runs we surface the `$["Node"]` accessors so
 * authors can see which node references are available. Skips internal keys
 * (the CEL rewrite alias and the `$` map itself, which we expose via derived
 * rows) to keep the list focused on user-visible fields.
 */
const INTERNAL_PREVIEW_KEYS = new Set(["__runNodes__"]);

function previewableFields(value: unknown): Array<{ key: string; preview: string }> {
  if (!value || typeof value !== "object") {
    return [{ key: "", preview: shortPreviewString(value) }];
  }
  const record = value as Record<string, unknown>;
  const out: Array<{ key: string; preview: string }> = [];
  for (const key of Object.keys(record)) {
    if (INTERNAL_PREVIEW_KEYS.has(key)) continue;
    if (key === "$") {
      const nodes = record.$ as Record<string, unknown> | undefined;
      if (nodes && typeof nodes === "object") {
        for (const nodeKey of Object.keys(nodes)) {
          const accessor = `$[${JSON.stringify(nodeKey)}].data`;
          out.push({ key: accessor, preview: shortPreviewString((nodes[nodeKey] as { data?: unknown })?.data) });
        }
      }
      continue;
    }
    out.push({ key, preview: shortPreviewString(record[key]) });
  }
  return out.slice(0, 12);
}

function shortPreviewString(value: unknown): string {
  if (value === null || value === undefined) return "-";
  if (typeof value === "string") return value.length > 80 ? `${value.slice(0, 80)}…` : value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    const json = JSON.stringify(value);
    return json.length > 80 ? `${json.slice(0, 80)}…` : json;
  } catch {
    return "[unserializable]";
  }
}
